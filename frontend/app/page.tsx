"use client";

import { useEffect, useState, useCallback, useRef } from "react";

import ThemeToggle from "./components/common/ThemeToggle";
import GameScreen from "./components/game/GameScreen";
import CategoryScreen from "./components/screens/CategoryScreen";
import JoinScreen from "./components/screens/JoinScreen";
import LoadingScreen from "./components/screens/LoadingScreen";
import RpsRevealScreen from "./components/screens/RpsRevealScreen";
import RpsScreen from "./components/screens/RpsScreen";
import WaitingScreen from "./components/screens/WaitingScreen";
import type { Category, GamePhase, GameState, ThemeMode, Tile } from "./types";

interface WSMessage {
  type: string;
  payload: unknown;
}

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL;
const WS_URL =
  process.env.NEXT_PUBLIC_WS_URL ||
  (backendUrl
    ? `${backendUrl.replace(/^http/, "ws").replace(/\/$/, "")}/ws`
    : "ws://localhost:8080/ws");

export default function Home() {
  const wsRef = useRef<WebSocket | null>(null);
  const pushSoundRef = useRef<HTMLAudioElement | null>(null);
  const shuffleSoundRef = useRef<HTMLAudioElement | null>(null);
  const lastBoardSignatureRef = useRef<string | null>(null);
  const lastPhaseRef = useRef<GamePhase | null>(null);
  const rpsRevealTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const winRevealTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const winContentTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastWinSignatureRef = useRef<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [playerId, setPlayerId] = useState<number | null>(null);
  const [playerName, setPlayerName] = useState("");
  const [nameInput, setNameInput] = useState("");
  const [playWithBot, setPlayWithBot] = useState(false);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [rpsChoice, setRpsChoice] = useState<string | null>(null);
  const [showRpsReveal, setShowRpsReveal] = useState(false);
  const [showWinOverlay, setShowWinOverlay] = useState(false);
  const [showWinContent, setShowWinContent] = useState(false);
  const [theme, setTheme] = useState<ThemeMode>("light");
  const [categoryChoice, setCategoryChoice] = useState<Category | null>(null);

  const sendMessage = useCallback((type: string, payload: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, payload }));
    } else {
      console.error("WebSocket is not open");
      location.reload();
    }
  }, []);

  useEffect(() => {
    if (typeof Audio === "undefined") return;
    //const audio = new Audio("/lego-breaking.mp3");
    const audio = new Audio("/lego_shuffle.m4a");
    console.log("Loading push sound...");
    audio.preload = "auto";
    audio.volume = 1;
    audio.load();
    pushSoundRef.current = audio;
  }, []);

  useEffect(() => {
    if (typeof Audio === "undefined") return;
    const audio = new Audio("/shuffle.m4a");
    audio.preload = "auto";
    audio.loop = true;
    audio.volume = 0.6;
    audio.load();
    shuffleSoundRef.current = audio;
  }, []);

  const playPushSound = useCallback(() => {
    const baseAudio = pushSoundRef.current;
    if (!baseAudio) return;
    const audio = baseAudio.cloneNode(true) as HTMLAudioElement;
    audio.volume = baseAudio.volume;
    audio.currentTime = 0;
    void audio.play().catch(() => undefined);
  }, []);

  const playShuffleSound = useCallback(() => {
    const audio = shuffleSoundRef.current;
    if (!audio) return;
    audio.currentTime = 0;
    void audio.play().catch(() => undefined);
  }, []);

  const stopShuffleSound = useCallback(() => {
    const audio = shuffleSoundRef.current;
    if (!audio) return;
    audio.pause();
    audio.currentTime = 0;
  }, []);

  const getBoardSignature = (state: GameState) =>
    state.players
      .map((player) => player.board.map((row) => row.map((tile) => tile.id).join(",")).join(";"))
      .join("|");

  useEffect(() => {
    if (!gameState) return;
    if (gameState.phase !== "playing" && gameState.phase !== "gameover") {
      lastBoardSignatureRef.current = null;
      return;
    }
    const signature = getBoardSignature(gameState);
    if (lastBoardSignatureRef.current && lastBoardSignatureRef.current !== signature) {
      playPushSound();
    }
    lastBoardSignatureRef.current = signature;
  }, [gameState, playPushSound]);

  useEffect(() => {
    if (!gameState || gameState.phase !== "gameover") {
      setShowWinOverlay(false);
      setShowWinContent(false);
      lastWinSignatureRef.current = null;
      if (winRevealTimeoutRef.current) {
        clearTimeout(winRevealTimeoutRef.current);
        winRevealTimeoutRef.current = null;
      }
      if (winContentTimeoutRef.current) {
        clearTimeout(winContentTimeoutRef.current);
        winContentTimeoutRef.current = null;
      }
      return;
    }
    const signature = getBoardSignature(gameState);
    if (lastWinSignatureRef.current !== signature) {
      lastWinSignatureRef.current = signature;
      if (winRevealTimeoutRef.current) {
        clearTimeout(winRevealTimeoutRef.current);
      }
      winRevealTimeoutRef.current = setTimeout(() => {
        setShowWinOverlay(true);
        winRevealTimeoutRef.current = null;
      }, 900);
      return;
    }
    if (!showWinOverlay && !winRevealTimeoutRef.current) {
      winRevealTimeoutRef.current = setTimeout(() => {
        setShowWinOverlay(true);
        winRevealTimeoutRef.current = null;
      }, 900);
    }
  }, [gameState, showWinOverlay]);

  useEffect(() => {
    if (!showWinOverlay) {
      setShowWinContent(false);
      if (winContentTimeoutRef.current) {
        clearTimeout(winContentTimeoutRef.current);
        winContentTimeoutRef.current = null;
      }
      return;
    }
    if (winContentTimeoutRef.current) {
      clearTimeout(winContentTimeoutRef.current);
    }
    winContentTimeoutRef.current = setTimeout(() => {
      setShowWinContent(true);
      winContentTimeoutRef.current = null;
    }, 1400);
    return () => {
      if (winContentTimeoutRef.current) {
        clearTimeout(winContentTimeoutRef.current);
        winContentTimeoutRef.current = null;
      }
    };
  }, [showWinOverlay]);

  useEffect(() => {
    if (!gameState) return;
    if (gameState.phase !== "rps") {
      setRpsChoice(null);
      return;
    }
    if (playerId === null) return;
    const serverChoice = gameState.rpsChoices?.[playerId];
    if (serverChoice) {
      setRpsChoice(serverChoice);
      return;
    }
    if (gameState.statusMessage.toLowerCase().includes("tie")) {
      setRpsChoice(null);
    }
  }, [gameState, playerId]);

  useEffect(() => {
    if (!gameState) {
      stopShuffleSound();
      return;
    }
    const previousPhase = lastPhaseRef.current;
    lastPhaseRef.current = gameState.phase;
    if (gameState.phase !== "playing") {
      setShowRpsReveal(false);
      stopShuffleSound();
      if (rpsRevealTimeoutRef.current) {
        clearTimeout(rpsRevealTimeoutRef.current);
        rpsRevealTimeoutRef.current = null;
      }
      return;
    }
    const choices = gameState.rpsChoices ?? {};
    const ready = Object.keys(choices).length === 2;
    if (previousPhase !== "playing" && ready) {
      setShowRpsReveal(true);
      playShuffleSound();
      if (rpsRevealTimeoutRef.current) {
        clearTimeout(rpsRevealTimeoutRef.current);
      }
      rpsRevealTimeoutRef.current = setTimeout(() => {
        setShowRpsReveal(false);
        stopShuffleSound();
        rpsRevealTimeoutRef.current = null;
      }, 4000);
    }
  }, [gameState, playShuffleSound, stopShuffleSound]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const stored = window.localStorage.getItem("theme");
    const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
    const initial = stored === "light" || stored === "dark" ? stored : prefersDark ? "dark" : "light";
    setTheme(initial);
    document.documentElement.setAttribute("data-theme", initial);
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    document.documentElement.setAttribute("data-theme", theme);
    window.localStorage.setItem("theme", theme);
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme((current) => (current === "dark" ? "light" : "dark"));
  }, []);

  useEffect(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onclose = () => {
      setConnected(false);
    };

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        if (msg.type === "joined") {
          const data = msg.payload as { playerId: number; roomId: string };
          setPlayerId(data.playerId);
        } else if (msg.type === "gameState") {
          const state = msg.payload as GameState;
          setGameState(state);
          if (state.phase !== "category") {
            setCategoryChoice(null);
          }
        }
      } catch (e) {
        console.error("WS message parse error:", e);
      }
    };

    return () => {
      ws.close();
    };
  }, []);

  const handleJoin = useCallback(() => {
    if (nameInput.trim()) {
      setPlayerName(nameInput.trim());
      sendMessage("join", { name: nameInput.trim(), withBot: playWithBot });
    }
  }, [nameInput, playWithBot, sendMessage]);

  const handleRPS = useCallback(
    (choice: string) => {
      if (gameState?.phase === "rps") {
        setRpsChoice(choice);
        sendMessage("rps", { choice });
      }
    },
    [gameState?.phase, sendMessage]
  );

  const handleCategorySelect = useCallback(
    (category: Category) => {
      if (gameState?.phase === "category") {
        setCategoryChoice(category);
        sendMessage("category", { category });
      }
    },
    [gameState?.phase, sendMessage]
  );

  const handlePush = useCallback(
    (column: number) => {
      if (gameState?.phase === "playing" && playerId === gameState.currentTurn) {
        sendMessage("push", { column });
      }
    },
    [gameState?.phase, gameState?.currentTurn, playerId, sendMessage]
  );

  const handleReset = useCallback(() => {
    sendMessage("reset", {});
  }, [sendMessage]);

  const currentPlayer = gameState?.players.find((p) => p.id === playerId);
  const opponent = gameState?.players.find((p) => p.id !== playerId);
  const isMyTurn = gameState?.currentTurn === playerId;
  const rpsChoices = gameState?.rpsChoices ?? {};
  const rpsReady = Object.keys(rpsChoices).length === 2;
  const shouldShowRpsReveal =
    gameState?.phase === "playing" &&
    rpsReady &&
    (showRpsReveal || lastPhaseRef.current !== "playing");
  const opponentId = playerId === null ? null : playerId === 0 ? 1 : 0;
  const myRpsChoice = playerId === null ? undefined : rpsChoices[playerId];
  const opponentRpsChoice = opponentId === null ? undefined : rpsChoices[opponentId];
  const themeToggle = (
    <div className="fixed right-4 top-4 z-50">
      <ThemeToggle theme={theme} onToggle={toggleTheme} />
    </div>
  );
  const playerKey = playerId !== null ? String(playerId) : "";
  const categoryChoices = gameState?.categoryChoices ?? {};
  const myCategoryChoice = playerKey ? categoryChoices[playerKey] : undefined;

  if (!playerName) {
    return (
      <JoinScreen
        themeToggle={themeToggle}
        nameInput={nameInput}
        onNameChange={setNameInput}
        onJoin={handleJoin}
        connected={connected}
        playWithBot={playWithBot}
        onPlayWithBotChange={setPlayWithBot}
      />
    );
  }

  if (shouldShowRpsReveal) {
    return (
      <RpsRevealScreen
        themeToggle={themeToggle}
        opponentName={opponent?.name}
        myChoice={myRpsChoice}
        opponentChoice={opponentRpsChoice}
      />
    );
  }

  if (!gameState) {
    return <LoadingScreen themeToggle={themeToggle} message="Waiting for game state..." />;
  }

  if (gameState.phase === "waiting") {
    return <WaitingScreen themeToggle={themeToggle} statusMessage={gameState.statusMessage} />;
  }

  if (gameState.phase === "category") {
    const categories = gameState.availableCategories ?? [];
    return (
      <CategoryScreen
        themeToggle={themeToggle}
        categories={categories}
        statusMessage={gameState.statusMessage}
        categoryChoices={categoryChoices}
        playerKey={playerKey}
        categoryChoice={categoryChoice}
        myCategoryChoice={myCategoryChoice}
        onSelect={handleCategorySelect}
      />
    );
  }

  if (gameState.phase === "rps") {
    const hasChosen = rpsChoice !== null;
    return (
      <RpsScreen
        themeToggle={themeToggle}
        statusMessage={gameState.statusMessage}
        rpsChoice={rpsChoice}
        hasChosen={hasChosen}
        onChoose={handleRPS}
      />
    );
  }

  return (
    <GameScreen
      themeToggle={themeToggle}
      gameState={gameState}
      currentPlayer={currentPlayer}
      opponent={opponent}
      isMyTurn={isMyTurn}
      playerId={playerId}
      showWinOverlay={showWinOverlay}
      showWinContent={showWinContent}
      onPush={handlePush}
      onReset={handleReset}
    />
  );
}
