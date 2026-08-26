(function () {
  function runReplace(file, slug, hooks) {
    if (!confirm("Replace this track's audio file? The current file will be deleted immediately; this can't be undone.")) {
      hooks.onCancel && hooks.onCancel();
      return;
    }

    hooks.onStart();

    fetch(`/admin/tracks/${slug}/replace/request`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        filename: file.name,
        content_type: file.type || "application/octet-stream",
      }),
    })
      .then((resp) => {
        if (!resp.ok) throw new Error("Failed to prepare replace");
        return resp.json();
      })
      .then(({ put_url }) => {
        return new Promise((resolve, reject) => {
          const xhr = new XMLHttpRequest();
          xhr.open("PUT", put_url);
          xhr.setRequestHeader("content-type", file.type || "application/octet-stream");
          xhr.upload.onprogress = (evt) => {
            if (evt.lengthComputable) hooks.onProgress((evt.loaded / evt.total) * 100);
          };
          xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error("Upload failed")));
          xhr.onerror = () => reject(new Error("Upload failed"));
          xhr.send(file);
        });
      })
      .then(() => {
        hooks.onProcessing();
        return pollStatus(slug);
      })
      .then(() => {
        location.reload();
      })
      .catch((err) => {
        hooks.onError(err.message || "Something went wrong");
      });
  }

  function pollStatus(slug) {
    return new Promise((resolve, reject) => {
      (function poll() {
        fetch(`/admin/upload/status/${slug}`)
          .then((r) => r.json())
          .then((data) => {
            if (data.status === "ready") return resolve();
            if (data.status === "failed") return reject(new Error("Processing failed"));
            setTimeout(poll, 2000);
          })
          .catch(reject);
      })();
    });
  }

  // Edit page: single dropzone.
  const editDropzone = document.getElementById("replace-dropzone");
  if (editDropzone) {
    const fileInput = document.getElementById("replace-file");
    const hint = document.getElementById("replace-dropzone-hint");
    const filenameEl = document.getElementById("replace-dropzone-filename");
    const progressWrap = document.getElementById("replace-progress-wrap");
    const progress = document.getElementById("replace-progress");
    const statusText = document.getElementById("replace-status-text");
    const errorEl = document.getElementById("replace-error");
    const slug = editDropzone.dataset.slug;

    editDropzone.addEventListener("click", () => fileInput.click());
    ["dragenter", "dragover"].forEach((evt) =>
      editDropzone.addEventListener(evt, (e) => {
        e.preventDefault();
        editDropzone.classList.add("drag-active");
      })
    );
    ["dragleave", "dragend"].forEach((evt) =>
      editDropzone.addEventListener(evt, () => editDropzone.classList.remove("drag-active"))
    );
    editDropzone.addEventListener("drop", (e) => {
      e.preventDefault();
      editDropzone.classList.remove("drag-active");
      const file = e.dataTransfer.files[0];
      if (file) startEditReplace(file);
    });
    fileInput.addEventListener("change", () => {
      const file = fileInput.files[0];
      if (file) startEditReplace(file);
    });

    function startEditReplace(file) {
      runReplace(file, slug, {
        onCancel() {
          fileInput.value = "";
        },
        onStart() {
          errorEl.hidden = true;
          filenameEl.textContent = file.name;
          hint.textContent = "File selected";
          progressWrap.hidden = false;
          statusText.textContent = "Uploading...";
          progress.value = 0;
        },
        onProgress(pct) {
          progress.value = pct;
        },
        onProcessing() {
          statusText.textContent = "Processing...";
          progress.removeAttribute("value");
        },
        onError(msg) {
          errorEl.textContent = msg;
          errorEl.hidden = false;
          progressWrap.hidden = true;
          fileInput.value = "";
        },
      });
    }
  }

  // Dashboard: one control per row.
  document.querySelectorAll(".row-replace-trigger").forEach((btn) => {
    const slug = btn.dataset.slug;
    const fileInput = document.querySelector(`.row-replace-file[data-slug="${slug}"]`);
    if (!fileInput) return;

    btn.addEventListener("click", () => fileInput.click());
    fileInput.addEventListener("change", () => {
      const file = fileInput.files[0];
      if (!file) return;

      const row = btn.closest(".grid-row");
      const statusEl = row ? row.querySelector(".status") : null;
      const originalStatus = statusEl ? statusEl.textContent : "";

      runReplace(file, slug, {
        onCancel() {
          fileInput.value = "";
        },
        onStart() {
          btn.disabled = true;
          if (statusEl) statusEl.textContent = "Uploading...";
        },
        onProgress(pct) {
          if (statusEl) statusEl.textContent = `Uploading ${Math.round(pct)}%...`;
        },
        onProcessing() {
          if (statusEl) statusEl.textContent = "Processing...";
        },
        onError(msg) {
          btn.disabled = false;
          fileInput.value = "";
          if (statusEl) statusEl.textContent = originalStatus;
          alert(msg);
        },
      });
    });
  });
})();
