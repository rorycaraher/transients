document.getElementById("upload-form").addEventListener("submit", async (e) => {
  e.preventDefault();

  const fileInput = document.getElementById("file");
  const file = fileInput.files[0];
  if (!file) return;

  const title = document.getElementById("title").value.trim();
  const expiresDays = document.getElementById("expires-days").value.trim();
  const downloadable = document.getElementById("downloadable").checked;

  const submitBtn = document.getElementById("submit-btn");
  const progressWrap = document.getElementById("progress-wrap");
  const progress = document.getElementById("progress");
  const statusText = document.getElementById("status-text");
  const errorEl = document.getElementById("error");

  errorEl.hidden = true;
  submitBtn.disabled = true;
  progressWrap.hidden = false;

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

    await pollStatus(slug);
  } catch (err) {
    errorEl.textContent = err.message || "Something went wrong";
    errorEl.hidden = false;
    progressWrap.hidden = true;
    submitBtn.disabled = false;
  }
});

async function pollStatus(slug) {
  const statusText = document.getElementById("status-text");
  while (true) {
    const resp = await fetch(`/admin/upload/status/${slug}`);
    const data = await resp.json();
    if (data.status === "ready") {
      document.getElementById("progress-wrap").hidden = true;
      const resultEl = document.getElementById("result");
      const link = document.getElementById("result-link");
      link.href = data.share_url;
      link.textContent = data.share_url;
      resultEl.hidden = false;
      return;
    }
    if (data.status === "failed") {
      throw new Error("Processing failed");
    }
    statusText.textContent = "Processing...";
    await new Promise((r) => setTimeout(r, 2000));
  }
}
