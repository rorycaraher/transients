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

// Diagnostics for an intermittent bug where playback halts mid-track with
// no visible error; these events would otherwise fail silently.
function mediaState() {
  return {
    currentTime: audio.currentTime,
    duration: audio.duration,
    networkState: audio.networkState,
    readyState: audio.readyState,
  };
}
audio.addEventListener("error", () => {
  const err = audio.error;
  console.error("[player] error", { code: err?.code, message: err?.message, ...mediaState() });
});
audio.addEventListener("stalled", () => {
  console.warn("[player] stalled", mediaState());
});
audio.addEventListener("waiting", () => {
  console.warn("[player] waiting", mediaState());
});

// Beacon exactly once per page load: replays/scrubbing within the same
// visit shouldn't inflate the count.
let playRecorded = false;
audio.addEventListener("play", () => {
  if (playRecorded) return;
  playRecorded = true;
  fetch(PLAYER_DATA.playUrl, { method: "POST", keepalive: true }).catch(() => {});
});

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
  seekBar.classList.add("is-dragging");
  seekBar.setPointerCapture(e.pointerId);
  seekFromEvent(e);
});
seekBar.addEventListener("pointermove", (e) => {
  if (dragging) seekFromEvent(e);
});
seekBar.addEventListener("pointerup", () => {
  dragging = false;
  seekBar.classList.remove("is-dragging");
});
seekBar.addEventListener("pointercancel", () => {
  dragging = false;
  seekBar.classList.remove("is-dragging");
});

const SKIP_SECONDS = 10;

function isEditableTarget(target) {
  if (!target) return false;
  if (target.isContentEditable) return true;
  return target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT";
}

document.addEventListener("keydown", (e) => {
  if (e.ctrlKey || e.metaKey || e.altKey) return;
  if (isEditableTarget(e.target)) return;

  switch (e.key.toLowerCase()) {
    case "k":
    case " ":
      e.preventDefault();
      if (audio.paused) audio.play();
      else audio.pause();
      break;
    case "j":
      if (!audio.duration) break;
      audio.currentTime = Math.max(0, audio.currentTime - SKIP_SECONDS);
      updateProgress();
      break;
    case "l":
      if (!audio.duration) break;
      audio.currentTime = Math.min(audio.duration, audio.currentTime + SKIP_SECONDS);
      updateProgress();
      break;
    default:
      if (e.key >= "0" && e.key <= "9") {
        if (!audio.duration) break;
        audio.currentTime = (Number(e.key) / 10) * audio.duration;
        updateProgress();
      }
      break;
  }
});
