const ws = WaveSurfer.create({
  container: "#waveform",
  waveColor: "#8f8b86",
  progressColor: "#9b8bc4",
  cursorColor: "#f2f0ec",
  url: PLAYER_DATA.audioUrl,
  peaks: [PLAYER_DATA.peaks],
  duration: PLAYER_DATA.duration,
});

function formatTime(seconds) {
  const minutes = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return `${minutes}:${secs.toString().padStart(2, "0")}`;
}

const playBtn = document.getElementById("play-btn");
const timeCurrent = document.getElementById("time-current");
const timeDuration = document.getElementById("time-duration");

timeDuration.textContent = formatTime(PLAYER_DATA.duration);

playBtn.addEventListener("click", () => ws.playPause());
ws.on("play", () => {
  playBtn.classList.add("is-playing");
  playBtn.setAttribute("aria-label", "Pause");
});
ws.on("pause", () => {
  playBtn.classList.remove("is-playing");
  playBtn.setAttribute("aria-label", "Play");
});
ws.on("timeupdate", (time) => {
  timeCurrent.textContent = formatTime(time);
});
