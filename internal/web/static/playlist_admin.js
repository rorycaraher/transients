// Drag-and-drop reordering for the Playlist edit page. Each row's ↑/↓ forms
// are the no-JS path to the same result; this only adds a faster gesture
// and posts the whole new order at once.
(function () {
  const list = document.getElementById("entries");
  if (!list) return;

  let dragged = null;

  list.addEventListener("dragstart", (e) => {
    const handle = e.target.closest(".drag-handle");
    if (!handle) return;
    dragged = handle.closest("[data-entry-id]");
    dragged.classList.add("is-dragging");
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", dragged.dataset.entryId);
    e.dataTransfer.setDragImage(dragged, 0, 0);
  });

  list.addEventListener("dragover", (e) => {
    if (!dragged) return;
    e.preventDefault();
    const row = e.target.closest("[data-entry-id]");
    if (!row || row === dragged) return;
    const rect = row.getBoundingClientRect();
    const after = e.clientY > rect.top + rect.height / 2;
    list.insertBefore(dragged, after ? row.nextSibling : row);
  });

  list.addEventListener("drop", (e) => e.preventDefault());

  list.addEventListener("dragend", () => {
    if (!dragged) return;
    dragged.classList.remove("is-dragging");
    dragged = null;

    const ids = Array.from(list.querySelectorAll("[data-entry-id]"), (row) => Number(row.dataset.entryId));
    fetch(list.dataset.orderUrl, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ entry_ids: ids }),
    })
      .then((resp) => {
        if (!resp.ok) throw new Error("reorder failed");
      })
      .catch(() => {})
      .finally(() => location.reload());
  });
})();
