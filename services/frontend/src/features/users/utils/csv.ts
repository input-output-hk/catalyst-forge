export function exportUsersCsv(rows: Array<Record<string, unknown>>) {
  if (!rows || rows.length === 0) {
    return;
  }

  const header = Object.keys(rows[0] || {}).join(",");
  const body = rows
    .map((r) =>
      Object.values(r)
        .map((v) => `${String(v ?? "").replace(/"/g, '""')}`)
        .join(",")
    )
    .join("\n");
  const csv = `${header}\n${body}`;
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `users_export_${new Date().toISOString().replace(/[-:]/g, "").split(".")[0]}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}
