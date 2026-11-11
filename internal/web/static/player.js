const audio = document.getElementById("audio");
const tracks = PLAYER_DATA.tracks;
const isPlaylist = PLAYER_DATA.playlist;
let current = 0;
audio.src = tracks[current].audioUrl;

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

// Beacon at most once per Track per page load: replays/scrubbing within the
// same visit shouldn't inflate the count.
const playsRecorded = new Set();
audio.addEventListener("play", () => {
  if (playsRecorded.has(current)) return;
  playsRecorded.add(current);
  fetch(tracks[current].playUrl, { method: "POST", keepalive: true }).catch(() => {});
});

function setPlaying(playing) {
  playBtn.classList.toggle("is-playing", playing);
  playBtn.setAttribute("aria-label", playing ? "Pause" : "Play");
}
audio.addEventListener("play", () => setPlaying(true));
audio.addEventListener("pause", () => setPlaying(false));
audio.addEventListener("ended", () => {
  if (isPlaylist && current < tracks.length - 1) {
    loadTrack(current + 1, true);
  } else {
    setPlaying(false);
  }
});

const tracklistItems = isPlaylist ? document.querySelectorAll(".tracklist-item") : [];

function loadTrack(index, play) {
  current = index;
  audio.src = tracks[index].audioUrl;
  seekFill.style.width = "0%";
  timeCurrent.textContent = "0:00";
  timeDuration.textContent = "0:00";
  tracklistItems.forEach((item) => {
    item.classList.toggle("is-current", Number(item.dataset.index) === index);
  });
  if (play) audio.play();
}

const RESTART_THRESHOLD_SECONDS = 3;

// Advance preserves play/pause state; only the automatic Advance at the end
// of a Track (the "ended" handler) always continues playing.
function advance(step) {
  const wasPlaying = !audio.paused;
  if (step < 0 && audio.currentTime > RESTART_THRESHOLD_SECONDS) {
    audio.currentTime = 0;
    updateProgress();
    return;
  }
  const next = current + step;
  if (next < 0 || next >= tracks.length) return;
  loadTrack(next, wasPlaying);
}

document.querySelectorAll(".tracklist-select").forEach((btn) => {
  btn.addEventListener("click", () => loadTrack(Number(btn.dataset.index), true));
});

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

  if (isPlaylist && e.shiftKey) {
    const key = e.key.toLowerCase();
    if (key === "n" || key === "p") {
      advance(key === "n" ? 1 : -1);
      return;
    }
  }

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
