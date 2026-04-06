import { useMemo, useState } from "react";

import {
  bulkUpdateDocuments,
  completeContentUpload,
  createDocumentShareLink,
  detectDuplicates,
  getDocumentVersions,
  searchContent,
  setMergeFlag,
  startContentUpload,
  uploadContentChunk,
  type DocumentVersion,
  type DocumentSummary,
} from "../../api/endpoints";
import { AccessGate } from "../../auth/access-control";
import { useSessionStore } from "../../state/session-store";
import { buildUploadResumeKey } from "../../state/upload-resume-cache";

export function ContentLibraryPage() {
  const sessionUser = useSessionStore((s) => s.user);
  const role = useSessionStore((s) => s.user?.primaryRole ?? null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [results, setResults] = useState<DocumentSummary[]>([]);
  const [duplicateId, setDuplicateID] = useState("");
  const [selectedDocs, setSelectedDocs] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [documentIDForVersion, setDocumentIDForVersion] = useState("");
  const [title, setTitle] = useState("Uploaded Document");
  const [summary, setSummary] = useState("Uploaded from frontend");
  const [difficulty, setDifficulty] = useState(1);
  const [durationMinutes, setDurationMinutes] = useState(15);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [activeDocumentID, setActiveDocumentID] = useState("");
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [shareToken, setShareToken] = useState("");
  const [shareExpiresAt, setShareExpiresAt] = useState("");

  const resumeKey = useMemo(
    () =>
      file && sessionUser
        ? buildUploadResumeKey(
            sessionUser.tenantId,
            sessionUser.userId,
            file.name,
            file.size,
            documentIDForVersion || "new",
          )
        : "",
    [documentIDForVersion, file, sessionUser],
  );

  const runSearch = async () => {
    if (!query.trim()) {
      setError("Search query is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const items = await searchContent(query.trim(), 25);
      setResults(items);
      if (items.length === 0) {
        setStatus("No results");
      }
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const runDetectDuplicates = async () => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await detectDuplicates();
      setStatus(`Duplicate scan completed. Flagged: ${out.flagged}`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const runSetMergeFlag = async () => {
    if (!duplicateId.trim()) {
      setError("Duplicate ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await setMergeFlag(duplicateId.trim(), true);
      setStatus("Merge candidate flag updated");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const runBulkArchive = async () => {
    const ids = selectedDocs
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);
    if (ids.length === 0) {
      setError("Provide at least one document id");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await bulkUpdateDocuments({ document_ids: ids, archive: true });
      setStatus(`Archived ${ids.length} document(s)`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const runUpload = async (resume = true) => {
    if (!file) {
      setError("Choose a file to upload");
      return;
    }
    if (difficulty < 1 || difficulty > 5) {
      setError("Difficulty must be between 1 and 5");
      return;
    }
    if (durationMinutes < 5 || durationMinutes > 480) {
      setError("Duration must be between 5 and 480 minutes");
      return;
    }

    const chunkSize = 256 * 1024;
    const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      let uploadID = "";
      let startIndex = 0;
      if (resume && resumeKey) {
        const saved = localStorage.getItem(resumeKey);
        if (saved) {
          const parsed = JSON.parse(saved) as {
            uploadID: string;
            nextIndex: number;
          };
          uploadID = parsed.uploadID;
          startIndex = parsed.nextIndex;
        }
      }
      if (!uploadID) {
        const session = await startContentUpload({
          document_id: documentIDForVersion || undefined,
          file_name: file.name,
          mime_type: file.type || "application/octet-stream",
          total_chunks: totalChunks,
          chunk_size_bytes: chunkSize,
        });
        uploadID = session.upload_id;
        startIndex = 0;
      }

      for (let i = startIndex; i < totalChunks; i++) {
        const from = i * chunkSize;
        const to = Math.min((i + 1) * chunkSize, file.size);
        const buffer = await file.slice(from, to).arrayBuffer();
        const bytes = new Uint8Array(buffer);
        await withRetry(() => uploadContentChunk(uploadID, i, bytes), 3);
        if (resumeKey) {
          localStorage.setItem(
            resumeKey,
            JSON.stringify({ uploadID, nextIndex: i + 1 }),
          );
        }
        setUploadProgress(Math.round(((i + 1) / totalChunks) * 100));
      }

      const out = await completeContentUpload(uploadID, {
        title,
        summary,
        difficulty,
        duration_minutes: durationMinutes,
      });
      if (resumeKey) {
        localStorage.removeItem(resumeKey);
      }
      setActiveDocumentID(out.document_id);
      setStatus(
        `Upload completed. Document: ${out.document_id} (v${out.version_no})`,
      );
    } catch (e) {
      setError(messageFromError(e));
      setStatus("Upload stopped. You can retry to resume remaining chunks.");
    } finally {
      setLoading(false);
    }
  };

  const loadVersions = async () => {
    if (!activeDocumentID.trim()) {
      setError("Document ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const items = await getDocumentVersions(activeDocumentID.trim());
      setVersions(items);
      setStatus(`Loaded ${items.length} version(s)`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const createShare = async (version?: number) => {
    if (!activeDocumentID.trim()) {
      setError("Document ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await createDocumentShareLink(
        activeDocumentID.trim(),
        version,
      );
      setShareToken(out.token);
      setShareExpiresAt(out.expires_at);
      setStatus("Share link created (72h TTL)");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section>
      <h2>Content Library</h2>
      <p>Search content and operate duplicate/bulk workflows.</p>
      <p>Metadata constraints: difficulty 1-5, duration 5-480 minutes.</p>

      {error ? <p className="error">{error}</p> : null}
      {status ? <p>{status}</p> : null}

      <div className="login-panel">
        <h3>Search</h3>
        <div className="login-row">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="keyword"
          />
          <button onClick={runSearch} disabled={loading || !query.trim()}>
            {loading ? "Searching..." : "Search"}
          </button>
        </div>
        {results.length > 0 ? (
          <ul>
            {results.map((item) => (
              <li key={item.document_id}>
                {item.title} ({item.document_id})
              </li>
            ))}
          </ul>
        ) : (
          <p>Empty result set.</p>
        )}
      </div>

      <AccessGate
        role={role}
        permission="content.manage"
        fallback={<p>Read-only role. Upload actions disabled.</p>}
      >
        <div className="login-panel">
          <h3>Upload / Versioning</h3>
          <div className="login-row">
            <input
              type="file"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
            <input
              value={documentIDForVersion}
              onChange={(e) => setDocumentIDForVersion(e.target.value)}
              placeholder="existing document id (optional)"
            />
          </div>
          <div className="login-row">
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="title"
            />
            <input
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="summary"
            />
          </div>
          <div className="login-row">
            <input
              type="number"
              min={1}
              max={5}
              value={difficulty}
              onChange={(e) => setDifficulty(Number(e.target.value))}
              placeholder="difficulty"
            />
            <input
              type="number"
              min={5}
              max={480}
              value={durationMinutes}
              onChange={(e) => setDurationMinutes(Number(e.target.value))}
              placeholder="duration_minutes"
            />
            <button onClick={() => runUpload(true)} disabled={loading || !file}>
              {loading ? "Uploading..." : "Upload / Resume"}
            </button>
          </div>
          <p>Upload progress: {uploadProgress}%</p>
        </div>
      </AccessGate>

      <div className="login-panel">
        <h3>Document Versions / Share</h3>
        <div className="login-row">
          <input
            value={activeDocumentID}
            onChange={(e) => setActiveDocumentID(e.target.value)}
            placeholder="document id"
          />
          <button
            onClick={loadVersions}
            disabled={loading || !activeDocumentID.trim()}
          >
            {loading ? "Loading..." : "Load Versions"}
          </button>
          <button
            onClick={() => createShare()}
            disabled={loading || !activeDocumentID.trim()}
          >
            {loading ? "Creating..." : "Create Share Link"}
          </button>
        </div>
        {versions.length > 0 ? (
          <ul>
            {versions.map((version) => (
              <li key={version.document_version_id}>
                v{version.version_no} {version.file_name}{" "}
                <a
                  href={`/api/v1/content/documents/${encodeURIComponent(activeDocumentID)}/preview?version=${version.version_no}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  Preview
                </a>{" "}
                <a
                  href={`/api/v1/content/documents/${encodeURIComponent(activeDocumentID)}/download?version=${version.version_no}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  Watermarked Download
                </a>{" "}
                <button
                  onClick={() => createShare(version.version_no)}
                  disabled={loading}
                >
                  Share This Version
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <p>No versions loaded.</p>
        )}
        {shareToken ? (
          <p>
            Share token: {shareToken} | Expires:{" "}
            {new Date(shareExpiresAt).toLocaleString()}
          </p>
        ) : null}
      </div>

      <AccessGate
        role={role}
        permission="content.manage"
        fallback={<p>Read-only role. Management actions disabled.</p>}
      >
        <div className="login-panel">
          <h3>Duplicate Handling</h3>
          <div className="login-row">
            <button onClick={runDetectDuplicates} disabled={loading}>
              {loading ? "Running..." : "Detect Duplicates"}
            </button>
            <input
              value={duplicateId}
              onChange={(e) => setDuplicateID(e.target.value)}
              placeholder="duplicate id"
            />
            <button
              onClick={runSetMergeFlag}
              disabled={loading || !duplicateId.trim()}
            >
              {loading ? "Saving..." : "Set Merge Candidate"}
            </button>
          </div>
        </div>

        <div className="login-panel">
          <h3>Bulk Update</h3>
          <div className="login-row">
            <input
              value={selectedDocs}
              onChange={(e) => setSelectedDocs(e.target.value)}
              placeholder="document ids (comma separated)"
            />
            <button
              onClick={runBulkArchive}
              disabled={loading || !selectedDocs.trim()}
            >
              {loading ? "Submitting..." : "Archive Selected"}
            </button>
          </div>
        </div>
      </AccessGate>
    </section>
  );
}

async function withRetry<T>(
  fn: () => Promise<T>,
  attempts: number,
): Promise<T> {
  let lastError: unknown;
  for (let i = 0; i < attempts; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError;
}

function messageFromError(e: unknown): string {
  if (typeof e === "object" && e && "message" in e) {
    return String((e as { message: string }).message);
  }
  return "Request failed";
}
