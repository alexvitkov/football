package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const TICK_RATE = 16
const POST_GOAL_TIME = 2
const PLAYER_ACCEL = 1000
const PLAYER_DEACCEL = 700
const PLAYER_TURN_RATE = 5
const BALL_AIR_TIME = 1         // seconds
const BALL_AIR_TIME_BOOST = 1.5 // ball will be 1.5x faster than usual hit
const BALL_DRAG = 1.7
const PLAYER_HIT_FORCE = 1.15
const PLAYER_MIN_HIT_SPEED = 50

const PLAYER_BOOST_DURATION = 0.25
const PLAYER_BOOST_ACCEL = 3250
const PLAYER_BOOST_DEACCEL = 500
const PLAYER_BOOST_HIT_FORCE = 1.4
const PLAYER_BOOST_TURN_RATE = 3
const PLAYER_BOOST_COOLDOWN = 1

const BALL_RADIUS = 25

type BoostState int

const (
	BOOST_READY BoostState = iota
	BOOST_ACTIVE
	BOOST_COOLDOWN
)

type Input struct {
	Force bool
	Angle float64
	Mouse bool
}

type Player struct {
	sendMessageChannel chan interface{}

	Rigidbody *Rigidbody

	Input Input
	Angle float64
	Team  int

	boostState BoostState
	boostTime  float64
}

type PlayerPair struct {
	Player1 *Player
	Player2 *Player
}

type RigidbodyCollisionCooldown struct {
	Rb1       *Rigidbody
	Rb2       *Rigidbody
	Remaining int
}

type Rigidbody struct {
	Position Vector2
	Radius   float64
	Velocity Vector2
	Mass     float64
	Drag     float64
	Force    float64
}

type GameState struct {
	TotalPlayers    int
	Players         []*Player
	Width           float64
	Height          float64
	Ball            *Rigidbody
	Goals           []Rect
	TickerPaused    bool
	BallAirTimeLeft float64
	LastTick        time.Time
	Score1          int
	Score2          int
	PostGoalTime    float64

	StaticColliders []Rect
	Rigidbodies     []*Rigidbody
	RbCollCD        []RigidbodyCollisionCooldown
}

type Handshake struct {
	GameId       string
	Team         int
	TotalPlayers int
}

type HelloMessage struct {
	MessageID        int32     `json:"msgId"` // 1
	PlayerPositions  []Vector2 `json:"playerPositions"`
	PlayerAngles     []float64 `json:"playerAngles"`
	Teams            []int     `json:"teams"`
	LocalPlayerIndex int       `json:"localPlayerIndex"`
	FieldWidth       float64   `json:"fieldWidth"`
	FieldHeight      float64   `json:"fieldHeight"`
	StaticColliders  []Rect    `json:"staticColliders"`
	Ball             Vector2   `json:"ball"`
	Goals            []Rect    `json:"goals"`
}

type PlayerJoinedMessage struct {
	MessageID int32   `json:"msgId"` // 2
	Position  Vector2 `json:"position"`
	Angle     float64 `json:"angle"`
	Team      int     `json:"team"`
}

type PlayerTick struct {
	PlayerID int     `json:"p"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Angle    float64 `json:"a"`
	Force    bool    `json:"force"`
}

type TickMessage struct {
	MessageID int32        `json:"msgId"` // 3
	Players   []PlayerTick `json:"pt"`
	Ball      Vector2      `json:"ball"`
}

type WriteTextMessage struct {
	MessageID int32  `json:"msgId"` // 4
	Message   string `json:"message"`
}

var games = map[string]*GameState{}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(req *http.Request) bool { return true },
}

func (self *Player) IfBoosted(if_true float64, if_false float64) float64 {
	if self.boostState == BOOST_ACTIVE {
		return if_true
	}
	return if_false
}

func newGame() *GameState {
	var fieldWidth float64 = 1000
	var fieldHeight float64 = 700

	var goalDepth float64 = 60
	var goalGap float64 = 200
	var wallWidth float64 = 200

	// goalTopY := fieldHeight/2 - goalGap/2 - goalWidth
	// goalBottomY := fieldHeight/2 + goalGap/2
	goalTopY := (fieldHeight - goalGap) / 2
	// goalBottomY := goalTopY + goalGap

	ball := &Rigidbody{
		Position: Vector2{fieldWidth / 2, fieldHeight / 2},
		Radius:   15,
		Drag:     0.98,
		Mass:     10,
		Force:    1,
	}

	return &GameState{
		Width:  fieldWidth,
		Height: fieldHeight,

		StaticColliders: []Rect{
			// goals
			{X: 0, Y: 0, Width: goalDepth, Height: goalTopY},
			{X: 0, Y: goalTopY + goalGap, Width: goalDepth, Height: goalTopY},
			{X: fieldWidth - goalDepth, Y: 0, Width: goalDepth, Height: goalTopY},
			{X: fieldWidth - goalDepth, Y: goalTopY + goalGap, Width: goalDepth, Height: goalTopY},

			// field walls
			{X: 0, Y: -wallWidth, Width: fieldWidth, Height: wallWidth},
			{X: 0, Y: fieldHeight, Width: fieldWidth, Height: wallWidth},
			{X: -wallWidth, Y: 0, Width: wallWidth, Height: fieldHeight},
			{X: fieldWidth, Y: 0, Width: wallWidth, Height: fieldHeight},
		},
		Ball:        ball,
		Rigidbodies: []*Rigidbody{ball},
		Goals: []Rect{
			{X: 0, Y: goalTopY, Width: goalDepth - BALL_RADIUS, Height: goalGap},
			{X: fieldWidth - goalDepth + BALL_RADIUS, Y: goalTopY, Width: goalDepth - BALL_RADIUS, Height: goalGap},
		},
		LastTick: time.Now(),
	}
}

func sendHello(game *GameState, index int) {
	positions := []Vector2{}
	angles := []float64{}
	teams := []int{}

	for _, player := range game.Players {
		positions = append(positions, player.Rigidbody.Position)
		angles = append(angles, player.Angle)
		teams = append(teams, player.Team)
	}

	game.Players[index].sendMessageChannel <- HelloMessage{
		MessageID:        1,
		PlayerPositions:  positions,
		PlayerAngles:     angles,
		Teams:            teams,
		LocalPlayerIndex: index,
		FieldWidth:       game.Width,
		FieldHeight:      game.Height,
		StaticColliders:  game.StaticColliders,
		Ball:             game.Ball.Position,
		Goals:            game.Goals,
	}
}
func countdown(game *GameState, callback func()) {
	for i := 3; i >= 0; i-- {
		for _, player := range game.Players {
			player.sendMessageChannel <- WriteTextMessage{
				MessageID: 4,
				Message:   fmt.Sprint(i),
			}
		}

		if i != 0 {
			time.Sleep(1 * time.Second)
		}
	}

	if callback != nil {
		callback()
	}
}

func simulatePhysics(game *GameState, dt float64) {
	for _, r := range game.Rigidbodies {
		r.Position.Add(r.Velocity.Multiplied(dt))
		r.Velocity.Multiply(r.Drag)
	}

	n := 0
	for _, coll := range game.RbCollCD {
		if coll.Remaining >= 0 {
			game.RbCollCD[n] = coll
			game.RbCollCD[n].Remaining--
			n++
		}
	}
	game.RbCollCD = game.RbCollCD[:n]

	for iter := 0; iter < 3; iter++ {
		for i1 := 0; i1 < len(game.Rigidbodies)-1; i1++ {
			for i2 := i1 + 1; i2 < len(game.Rigidbodies); i2++ {

				r1 := game.Rigidbodies[i1]
				r2 := game.Rigidbodies[i2]
				if r1.Mass < r2.Mass {
					tmp := r1
					r1 = r2
					r2 = tmp
				}

				skip := false
				for _, coll := range game.RbCollCD {
					if coll.Rb1 == r1 && coll.Rb2 == r2 {
						fmt.Println(coll)
						skip = true
					}
				}

				dist := Distance(r1.Position, r2.Position)
				wantedDist := r1.Radius + r2.Radius

				if dist < wantedDist {
					delta := r1.Position
					delta.Subtract(r2.Position)
					delta.Normalize()

					// Resolve collision
					r1.Position.Add(delta.Multiplied((dist - wantedDist) * -.5))
					r2.Position.Add(delta.Multiplied((dist - wantedDist) * .5))

					if !skip {
						game.RbCollCD = append(game.RbCollCD, RigidbodyCollisionCooldown{
							Rb1:       r1,
							Rb2:       r2,
							Remaining: 5,
						})

						factor := r1.Mass / (r1.Mass + r2.Mass)

						d1 := delta.Multiplied(-r1.Velocity.Magnitude() * factor * r1.Force)
						d2 := delta.Multiplied(-r2.Velocity.Magnitude() * (factor - 1) * r2.Force)

						r1.Velocity.Add(d2)
						r2.Velocity.Add(d1)
					}
				}
			}
		}
	}

	for _, r := range game.Rigidbodies {
		for _, rect := range game.StaticColliders {
			normal := collisionCircleRect(r, rect)
			r.Position.Add(normal)

			if normal.X != 0 || normal.Y != 0 {
				originalSpeed := r.Velocity.Magnitude()
				reflectionAngle := ReflectionAngle(r.Velocity.Angle(), normal.Angle())

				r.Velocity = Vector2{
					math.Cos(reflectionAngle),
					math.Sin(reflectionAngle),
				}

				r.Velocity.Multiply(originalSpeed)
			}

		}
	}
}

func startGame(game *GameState) {
	countdown(game, nil)
	ticker := time.NewTicker(TICK_RATE * time.Millisecond)

	go func() {
		game.LastTick = time.Now()

		for {
			now := <-ticker.C
			dt := float64(now.Sub(game.LastTick).Milliseconds()) / 1000
			game.LastTick = now

			if dt < 0.001 || dt > 0.5 {
				continue
			}

			if game.TickerPaused {
				continue
			}

			if game.PostGoalTime > 0 {
				game.PostGoalTime -= dt
				if game.PostGoalTime <= 0 {
					game.PostGoalTime = 0
					resetGame(game)
					game.TickerPaused = true
					countdown(game, func() {
						game.TickerPaused = false
						game.LastTick = time.Now()
					})
					continue
				}
			}

			if game.BallAirTimeLeft > 0 {
				game.BallAirTimeLeft -= dt
			}
			if game.BallAirTimeLeft < 0 {
				game.BallAirTimeLeft = 0
			}

			for _, player := range game.Players {

				// player boost time calculatinos
				if player.boostState != BOOST_READY {
					player.boostTime -= dt

					if player.boostTime <= 0 {
						if player.boostState == BOOST_ACTIVE {
							// exit boost
							player.boostState = BOOST_COOLDOWN
							player.boostTime = PLAYER_BOOST_COOLDOWN
							player.Rigidbody.Force = PLAYER_HIT_FORCE
						} else if player.boostState == BOOST_COOLDOWN {
							// boost just got out of cooldown
							player.boostState = BOOST_READY
							player.boostTime = 0
						}
					}
				}

				if player.Input.Force && player.boostState == BOOST_READY {
					// enter boost
					player.boostState = BOOST_ACTIVE
					player.boostTime = PLAYER_BOOST_DURATION
					player.Rigidbody.Force = PLAYER_BOOST_HIT_FORCE
				}

				player.Angle = AngleLerp(player.Angle, player.Input.Angle, player.IfBoosted(PLAYER_BOOST_TURN_RATE, PLAYER_TURN_RATE)*dt)

				if player.Input.Mouse || player.boostState == BOOST_ACTIVE {
					delta := Vector2{
						math.Cos(player.Angle),
						math.Sin(player.Angle),
					}
					player.Rigidbody.Velocity.Add(delta.Multiplied(player.IfBoosted(350, 30)))
				} 
			}


			// GOAL CHECK
			if game.PostGoalTime == 0 {
				for team, goal := range game.Goals {
					d := collisionCircleRect(game.Ball, goal)

					if d.X != 0 || d.Y != 0 {
						// Has a goal
						game.PostGoalTime = POST_GOAL_TIME
						if team == 0 {
							game.Score2++
						} else {
							game.Score1++
						}
						for _, player := range game.Players {
							player.sendMessageChannel <- WriteTextMessage{
								MessageID: 4,
								Message:   fmt.Sprintf("%v:%v", game.Score1, game.Score2),
							}
						}
					}
				}
			}

			simulatePhysics(game, dt)

			// Send the new game state every tick
			changed := []PlayerTick{}
			for index, player := range game.Players {
				changed = append(changed, PlayerTick{
					PlayerID: index,
					X:        player.Rigidbody.Position.X,
					Y:        player.Rigidbody.Position.Y,
					Force:    player.Input.Force,
					Angle:    player.Angle,
				})
			}

			for _, player := range game.Players {
				player.sendMessageChannel <- TickMessage{
					MessageID: 3,
					Players:   changed,
					Ball:      game.Ball.Position,
				}
			}
		}
	}()
}

func initPlayerPositions(game *GameState) {
	team1 := 0
	team2 := 0
	playersPerTeam := game.TotalPlayers / 2

	for _, player := range game.Players {

		if player.Team == 0 {
			team1++
		} else {
			team2++
		}

		indexInTeam := team1

		if player.Team == 0 {
			player.Rigidbody.Position.X = 200
			player.Angle = 0
		} else {
			player.Rigidbody.Position.X = 800
			player.Angle = math.Pi
			indexInTeam = team2
		}
		player.Rigidbody.Position.Y = 350 + float64(indexInTeam)*80 - float64(playersPerTeam)*40
	}
}

func resetGame(game *GameState) {
	game.Ball.Velocity = Vector2{}
	game.Ball.Position = Vector2{game.Width / 2, game.Height / 2}

	changed := []PlayerTick{}

	for playerIndex, player := range game.Players {
		player.boostState = BOOST_READY

		initPlayerPositions(game)

		// TODO reset rigidbody velocities
		changed = append(changed, PlayerTick{
			PlayerID: playerIndex,
			X:        player.Rigidbody.Position.X,
			Y:        player.Rigidbody.Position.Y,
			Force:    player.Input.Force,
			Angle:    player.Angle,
		})
	}

	for _, player := range game.Players {
		player.sendMessageChannel <- TickMessage{
			MessageID: 3,
			Players:   changed,
			Ball:      game.Ball.Position,
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	var handshake Handshake
	conn.ReadJSON(&handshake)

	fmt.Printf("handshake: %v\n", handshake)

	game, exists := games[handshake.GameId]
	if !exists {
		game = newGame()
		games[handshake.GameId] = game
		game.TotalPlayers = handshake.TotalPlayers
	}

	player := &Player{
		sendMessageChannel: make(chan any, 100),
		Team:               handshake.Team,
		Rigidbody: &Rigidbody{
			Radius: BALL_RADIUS,
			Mass:   70,
			Drag:   0.90,
			Force:  PLAYER_HIT_FORCE,
		},
	}

	game.Rigidbodies = append(game.Rigidbodies, player.Rigidbody)

	index := len(game.Players)
	game.Players = append(game.Players, player)

	if len(game.Players) == handshake.TotalPlayers {
		go startGame(game)
	}

	initPlayerPositions(game)

	sendHello(game, index)

	for i, p := range game.Players {
		if i != index {
			p.sendMessageChannel <- PlayerJoinedMessage{
				MessageID: 2,
				Position:  player.Rigidbody.Position,
				Angle:     player.Angle,
				Team:      player.Team,
			}
		}
	}

	go func() {
		for {
			msg := <-player.sendMessageChannel
			conn.WriteJSON(msg)
		}
	}()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		if len(p) < 2 {
			continue
		}

		switch p[0] {
		case 'f':
			player.Input.Force = p[1] == 'd'
		case 'm':
			player.Input.Mouse = p[1] == 'd'
		case 'r':
			f, err := strconv.ParseFloat(string(p[1:]), 64)
			if err == nil {
				player.Input.Angle = f
			} else {
			}
			break
		}

	}
}

func main() {
	http.HandleFunc("/client", wsHandler)
	log.Fatal(http.ListenAndServe(":6006", nil))
}
