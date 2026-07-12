const ws = WaveSurfer.create({
  container: "#waveform",
  waveColor: "#4f4a85",
  progressColor: "#383351",
  url: PLAYER_DATA.audioUrl,
  peaks: [PLAYER_DATA.peaks],
  duration: PLAYER_DATA.duration,
});

const playBtn = document.getElementById("play-btn");
playBtn.addEventListener("click", () => ws.playPause());
ws.on("play", () => (playBtn.textContent = "Pause"));
ws.on("pause", () => (playBtn.textContent = "Play"));
