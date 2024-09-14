import { Application, Container, Graphics, Sprite, Text, Texture } from "pixi.js";
import { gsap } from "gsap";
import { PixiPlugin } from "gsap/all";
import * as PIXI from "pixi.js";

gsap.registerPlugin(PixiPlugin);
(window as any).PIXI = PIXI;

const playerTextures = [
  Texture.from("/games/football/player_blue.png"),
  Texture.from("/games/football/player_red.png"),
];
const pitchTexture = Texture.from("/games/football/pitch.png");

class Player extends Container {
  constructor(team) {
    super();

    const g = new Graphics();
    g.beginFill(0x000000);
    g.drawCircle(0, 0, 25);
    this.addChild(g);

    const sprite = new Sprite(playerTextures[team]);
    sprite.anchor.set(.5);
    sprite.scale.set(.23);
    this.addChild(sprite);
  }

  toggleRed(visible: boolean) {
  }

}

class Ball extends Container {
  constructor() {
    super();

    const g = new Graphics();
    g.beginFill(0xFF0000);
    g.drawCircle(0, 0, 15);
    this.addChild(g);
  }
}

let players: Player[] = [];
let localPlayer: Player;
let ball = new Ball;

interface Vector2 {
  x: number;
  y: number;
}

interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

enum MessageId {
  Hello = 1,
  PlayerJoined = 2,
  Tick = 3,
  WriteText = 4,
}

interface HelloMessage {
  msgId: MessageId.Hello,
  playerPositions: Vector2[],
  playerAngles: number[],
  teams: number[],
  localPlayerIndex: number,
  fieldWidth: number,
  fieldHeight: number,
  staticColliders: Rect[],
  goals: Rect[],
  ball: Vector2,
}

interface PlayerJoinedMessage {
  msgId: MessageId.PlayerJoined;
  team: number;
  position: Vector2;
  angle: number;
}

interface PlayerTick {
  p: number; // player id
  x: number;
  y: number;
  a: number; // angle
  force: boolean;
}

interface TickMessage {
  msgId: MessageId.Tick;
  pt: PlayerTick[];
  ball: Vector2,
}

interface WriteTextMessage {
  msgId: MessageId.WriteText;
  message: string;
}


type Message = HelloMessage | PlayerJoinedMessage | TickMessage | WriteTextMessage;

const app = new Application({
  width: 1000,
  height: 700,
  backgroundColor: 0x2e6344,
});

const ws = new WebSocket(GAME_WS);

ws.onmessage = m => {
  const msg = JSON.parse(m.data) as Message;

  switch (msg.msgId) {
    case MessageId.Hello: {
      initialize(msg);
      break;
    }

    case MessageId.PlayerJoined: {
      newPlayer(msg.team, msg.position, msg.angle);
      break;
    }

    case MessageId.Tick: {
      for (const playerTick of msg.pt) {
        const player = players[playerTick.p];
        player.x = playerTick.x;
        player.y = playerTick.y;
        player.rotation = playerTick.a;
        player.toggleRed(playerTick.force);
      }

      ball.position.set(msg.ball.x, msg.ball.y);
      break;
    }

    case MessageId.WriteText: {
      const text = new Text(msg.message, {
        fontSize: 100,
        dropShadow: true,
        dropShadowBlur: 5,
        dropShadowDistance: 2,
        fill: [
          "#dbfffe",
          "#ccffdb"
        ],
        fontWeight: "bold",
        lineJoin: "round",
        strokeThickness: 5
      });

      app.stage.addChild(text)
      text.x = 500;
      text.y = 300;
      text.anchor.set(0.5);

      gsap.to(text, {
        pixi: {
          y: "-=100",
          alpha: 0,
          scale: 0.7,
        },
        duration: 2,
      })

    }
  }
}

function initialize(msg: HelloMessage) {
  app.stage.addChild(new Sprite(pitchTexture));

  let index = 0;
  for (const pos of msg.playerPositions) {
    const player = newPlayer(msg.teams[index], pos, msg.playerAngles[index]);
    if (msg.localPlayerIndex == players.length - 1) localPlayer = player;
    index++;
  }

  app.stage.addChild(ball);
  ball.position.set(msg.ball.x, msg.ball.y);

  document.body.appendChild(app.view);
}

function newPlayer(team: number, pos: Vector2, angle: number) {
  const player = new Player(team);

  player.x = pos.x;
  player.y = pos.y;
  player.rotation = angle;

  players.push(player);

  app.stage.addChild(player);

  return player;
}

window.onkeydown = (msg: KeyboardEvent) => {
  switch (msg.key.toLowerCase()) {
    case 'w': { ws.send('wd'); break; }
    case 'a': { ws.send('ad'); break; }
    case 's': { ws.send('sd'); break; }
    case 'd': { ws.send('dd'); break; }
    case ' ': { ws.send('fd'); break; }
  }
}

window.onkeyup = (msg: KeyboardEvent) => {
  switch (msg.key.toLowerCase()) {
    case 'w': { ws.send('wu'); break; }
    case 'a': { ws.send('au'); break; }
    case 's': { ws.send('su'); break; }
    case 'd': { ws.send('du'); break; }
    case ' ': { ws.send('fu'); break; }
  }
}

window.onmousedown = (msg: MouseEvent) => {
  ws.send('md');
}

window.onmouseup = (msg: MouseEvent) => {
  ws.send('mu');
}

window.onmousemove = (msg: MouseEvent) => {
  const { x, y } = localPlayer.getGlobalPosition();
  const dx =  msg.clientX - x;
  const dy = msg.clientY - y;

  const angle = Math.atan2(dy, dx);
  ws.send('r' + angle)
}

window.onresize = updateSize;
updateSize();

function updateSize() {

  const offset = 20;
  const W = 1000 + offset*2;
  const H = 700 + offset*2;

  const scaleX = window.innerWidth / W;  
  const scaleY = window.innerHeight / H;  

  const scale = Math.min(scaleX, scaleY);

  app.view.width = W * scale;
  app.view.height = H * scale;

  app.renderer.resize(window.innerWidth, window.innerHeight);

  app.stage.scale.set(scale);


  if (scale == scaleX) {
    app.stage.x = offset*scale;
    app.stage.y = (window.innerHeight - 700*scale) / 2; 
  } else {
    app.stage.x = (window.innerWidth - 1000*scale) / 2;
    app.stage.y = offset*scale;
  }
}