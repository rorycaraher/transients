const form = document.getElementById("upload-form");
const fileInput = document.getElementById("file");
const dropzone = document.getElementById("dropzone");
const dropzoneHint = document.getElementById("dropzone-hint");
const dropzoneFilename = document.getElementById("dropzone-filename");
const submitBtn = document.getElementById("submit-btn");

dropzone.addEventListener("click", () => fileInput.click());

["dragenter", "dragover"].forEach((evt) =>
  dropzone.addEventListener(evt, (e) => {
    e.preventDefault();
    dropzone.classList.add("drag-active");
  })
);
["dragleave", "dragend"].forEach((evt) =>
  dropzone.addEventListener(evt, () => dropzone.classList.remove("drag-active"))
);
dropzone.addEventListener("drop", (e) => {
  e.preventDefault();
  dropzone.classList.remove("drag-active");
  const file = e.dataTransfer.files[0];
  if (!file) return;
  const dt = new DataTransfer();
  dt.items.add(file);
  fileInput.files = dt.files;
  onFileChosen();
});

fileInput.addEventListener("change", onFileChosen);

function onFileChosen() {
  const file = fileInput.files[0];
  dropzoneHint.textContent = file ? "File selected" : "Drag & drop or click to select file";
  dropzoneFilename.textContent = file ? file.name : "No file chosen";
  submitBtn.disabled = !file;
}

function resolvedTitle(file) {
  const title = document.getElementById("title").value.trim();
  return title || file.name;
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();

  const file = fileInput.files[0];
  if (!file) return;

  const title = document.getElementById("title").value.trim();
  const expiresDays = document.getElementById("expires-days").value.trim();
  const downloadable = document.getElementById("downloadable").checked;

  const progressWrap = document.getElementById("progress-wrap");
  const progress = document.getElementById("progress");
  const statusText = document.getElementById("status-text");
  const errorEl = document.getElementById("error");

  errorEl.hidden = true;
  form.hidden = true;
  progressWrap.hidden = false;
  statusText.textContent = "Uploading...";

  try {
    const requestResp = await fetch("/admin/upload/request", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        title: title || undefined,
        filename: file.name,
        content_type: file.type || "application/octet-stream",
        expires_in_days: expiresDays ? Number(expiresDays) : undefined,
        downloadable,
      }),
    });
    if (!requestResp.ok) throw new Error("Failed to prepare upload");
    const { slug, put_url } = await requestResp.json();

    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("PUT", put_url);
      xhr.setRequestHeader("content-type", file.type || "application/octet-stream");
      xhr.upload.onprogress = (evt) => {
        if (evt.lengthComputable) {
          progress.value = (evt.loaded / evt.total) * 100;
        }
      };
      xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error("Upload failed")));
      xhr.onerror = () => reject(new Error("Upload failed"));
      xhr.send(file);
    });

    statusText.textContent = "Processing...";
    progress.removeAttribute("value");

    await pollStatus(slug, resolvedTitle(file));
  } catch (err) {
    errorEl.textContent = err.message || "Something went wrong";
    errorEl.hidden = false;
    progressWrap.hidden = true;
    form.hidden = false;
  }
});

async function pollStatus(slug, title) {
  const statusText = document.getElementById("status-text");
  while (true) {
    const resp = await fetch(`/admin/upload/status/${slug}`);
    const data = await resp.json();
    if (data.status === "ready") {
      showResult(title, data.share_url);
      return;
    }
    if (data.status === "failed") {
      throw new Error("Processing failed");
    }
    statusText.textContent = "Processing...";
    await new Promise((r) => setTimeout(r, 2000));
  }
}

function showResult(title, shareUrl) {
  document.getElementById("progress-wrap").hidden = true;
  document.getElementById("result-title").textContent = title;
  document.getElementById("result-link-value").textContent = shareUrl;
  document.getElementById("result-view").href = shareUrl;
  document.getElementById("result").hidden = false;
}

document.getElementById("copy-btn").addEventListener("click", async () => {
  const shareUrl = document.getElementById("result-link-value").textContent;
  const copyBtn = document.getElementById("copy-btn");
  await navigator.clipboard.writeText(shareUrl);
  copyBtn.textContent = "Copied";
  setTimeout(() => (copyBtn.textContent = "Copy"), 1500);
});

document.getElementById("result-reset").addEventListener("click", () => {
  form.reset();
  onFileChosen();
  document.getElementById("result").hidden = true;
  form.hidden = false;
});
