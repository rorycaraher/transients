const audio = document.getElementById("audio");
audio.src = PLAYER_DATA.audioUrl;

const playBtn = document.getElementById("play-btn");
const seekBar = document.getElementById("seek-bar");
const seekFill = document.getElementById("seek-fill");
const timeCurrent = document.getElementById("time-current");
const timeDuration = document.getElementById("time-duration");

function formatTime(seconds) {
  if (!isFinite(seconds)) return "0:00";
  const minutes = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return `${minutes}:${secs.toString().padStart(2, "0")}`;
}

function updateProgress() {
  const pct = audio.duration ? (audio.currentTime / audio.duration) * 100 : 0;
  seekFill.style.width = `${pct}%`;
  timeCurrent.textContent = formatTime(audio.currentTime);
}

audio.addEventListener("loadedmetadata", () => {
  timeDuration.textContent = formatTime(audio.duration);
});
audio.addEventListener("timeupdate", updateProgress);

function setPlaying(playing) {
  playBtn.classList.toggle("is-playing", playing);
  playBtn.setAttribute("aria-label", playing ? "Pause" : "Play");
}
audio.addEventListener("play", () => setPlaying(true));
audio.addEventListener("pause", () => setPlaying(false));
audio.addEventListener("ended", () => setPlaying(false));

playBtn.addEventListener("click", () => {
  if (audio.paused) audio.play();
  else audio.pause();
});

function seekFromEvent(e) {
  if (!audio.duration) return;
  const rect = seekBar.getBoundingClientRect();
  const x = Math.min(Math.max(e.clientX - rect.left, 0), rect.width);
  audio.currentTime = (x / rect.width) * audio.duration;
  updateProgress();
}

let dragging = false;
seekBar.addEventListener("pointerdown", (e) => {
  dragging = true;
  seekBar.setPointerCapture(e.pointerId);
  seekFromEvent(e);
});
seekBar.addEventListener("pointermove", (e) => {
  if (dragging) seekFromEvent(e);
});
seekBar.addEventListener("pointerup", () => {
  dragging = false;
});
seekBar.addEventListener("pointercancel", () => {
  dragging = false;
});
