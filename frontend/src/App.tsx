import { useState, useEffect, useRef, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import "./App.css";
import {
  TakeScreenshot,
  GetScreenshots,
  GetScreenshotBase64,
  AnalyzeScreenshot,
  DeleteScreenshot,
  CheckOllamaStatus,
} from "../wailsjs/go/main/App";
import { main } from "../wailsjs/go/models";

interface OllamaStatus {
  available: boolean;
  modelLoaded: boolean;
  error?: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

function formatDate(isoString: string): string {
  const d = new Date(isoString);
  return d.toLocaleString();
}

function App() {
  const [screenshots, setScreenshots] = useState<main.ScreenshotInfo[]>([]);
  const [selectedScreenshot, setSelectedScreenshot] =
    useState<main.ScreenshotInfo | null>(null);
  const [previewDataUri, setPreviewDataUri] = useState<string>("");
  const [analysisResult, setAnalysisResult] =
    useState<main.AnalysisResult | null>(null);
  const [prompt, setPrompt] = useState(
    "Describe what you see in this screenshot in detail."
  );
  const [isCapturing, setIsCapturing] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [ollamaStatus, setOllamaStatus] = useState<OllamaStatus | null>(null);
  const [showThinking, setShowThinking] = useState(false);
  const [elapsedTime, setElapsedTime] = useState(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadScreenshots = useCallback(async () => {
    const list = await GetScreenshots();
    setScreenshots(list || []);
  }, []);

  useEffect(() => {
    loadScreenshots();
    CheckOllamaStatus().then((status: any) => {
      setOllamaStatus(status as OllamaStatus);
    });
  }, [loadScreenshots]);

  useEffect(() => {
    if (!selectedScreenshot) {
      setPreviewDataUri("");
      return;
    }
    GetScreenshotBase64(selectedScreenshot.filePath).then((uri) => {
      setPreviewDataUri(uri);
    });
  }, [selectedScreenshot]);

  useEffect(() => {
    if (isAnalyzing) {
      setElapsedTime(0);
      timerRef.current = setInterval(() => {
        setElapsedTime((prev) => prev + 1);
      }, 1000);
    } else {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [isAnalyzing]);

  const handleCapture = async (mode: string) => {
    setIsCapturing(true);
    try {
      const result = await TakeScreenshot(mode);
      if (result.success) {
        await loadScreenshots();
        const list = await GetScreenshots();
        if (list && list.length > 0) {
          setSelectedScreenshot(list[0]);
        }
      } else {
        alert("Screenshot failed: " + result.error);
      }
    } finally {
      setIsCapturing(false);
    }
  };

  const handleAnalyze = async () => {
    if (!selectedScreenshot) return;
    setIsAnalyzing(true);
    setAnalysisResult(null);
    try {
      const result = await AnalyzeScreenshot(
        selectedScreenshot.filePath,
        prompt
      );
      setAnalysisResult(result);
    } finally {
      setIsAnalyzing(false);
    }
  };

  const handleDelete = async (
    e: React.MouseEvent,
    screenshot: main.ScreenshotInfo
  ) => {
    e.stopPropagation();
    const result = await DeleteScreenshot(screenshot.filePath);
    if (result.success) {
      if (selectedScreenshot?.filePath === screenshot.filePath) {
        setSelectedScreenshot(null);
        setAnalysisResult(null);
      }
      await loadScreenshots();
    }
  };

  return (
    <div className="app">
      {/* Header */}
      <header className="header">
        <h1 className="header-title">Screenshot Analyzer</h1>
        <div className="ollama-status">
          <span
            className={`status-dot ${
              ollamaStatus?.available && ollamaStatus?.modelLoaded
                ? "status-ok"
                : "status-err"
            }`}
          />
          <span className="status-text">
            {!ollamaStatus
              ? "Checking..."
              : ollamaStatus.available && ollamaStatus.modelLoaded
              ? "Ollama Ready"
              : ollamaStatus.available
              ? "Model not found"
              : "Ollama Offline"}
          </span>
        </div>
      </header>

      {/* Capture Bar */}
      <div className="capture-bar">
        <button
          className="capture-btn"
          onClick={() => handleCapture("full")}
          disabled={isCapturing}
        >
          {isCapturing ? "Capturing..." : "Full Screen"}
        </button>
        <button
          className="capture-btn"
          onClick={() => handleCapture("selection")}
          disabled={isCapturing}
        >
          Selection
        </button>
        <button
          className="capture-btn"
          onClick={() => handleCapture("focused")}
          disabled={isCapturing}
        >
          Active Window
        </button>
      </div>

      {/* Main Content */}
      <div className="main-content">
        {/* Left Panel - Gallery */}
        <aside className="gallery-panel">
          <h2 className="panel-title">
            Screenshots ({screenshots.length})
          </h2>
          <div className="gallery-list">
            {screenshots.length === 0 && (
              <p className="empty-text">No screenshots yet. Capture one above.</p>
            )}
            {screenshots.map((s) => (
              <div
                key={s.filePath}
                className={`gallery-item ${
                  selectedScreenshot?.filePath === s.filePath ? "selected" : ""
                }`}
                onClick={() => setSelectedScreenshot(s)}
              >
                <div className="gallery-item-info">
                  <span className="gallery-item-name">{s.fileName}</span>
                  <span className="gallery-item-meta">
                    {formatSize(s.size)} &middot; {formatDate(s.modTime)}
                  </span>
                </div>
                <button
                  className="delete-btn"
                  onClick={(e) => handleDelete(e, s)}
                  title="Delete"
                >
                  &times;
                </button>
              </div>
            ))}
          </div>
        </aside>

        {/* Right Panel - Preview & Analysis */}
        <main className="detail-panel">
          {!selectedScreenshot ? (
            <div className="empty-state">
              <p>Select a screenshot from the gallery or capture a new one.</p>
            </div>
          ) : (
            <>
              {/* Image Preview */}
              <div className="preview-container">
                {previewDataUri ? (
                  <img
                    src={previewDataUri}
                    alt={selectedScreenshot.fileName}
                    className="preview-image"
                  />
                ) : (
                  <div className="preview-loading">Loading preview...</div>
                )}
              </div>

              {/* Prompt & Analyze */}
              <div className="analyze-section">
                <textarea
                  className="prompt-input"
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder="Enter your prompt..."
                  rows={3}
                />
                <button
                  className="analyze-btn"
                  onClick={handleAnalyze}
                  disabled={isAnalyzing || !ollamaStatus?.modelLoaded}
                >
                  {isAnalyzing ? (
                    <>
                      <span className="spinner" /> Analyzing... ({elapsedTime}s)
                    </>
                  ) : (
                    "Analyze"
                  )}
                </button>
              </div>

              {/* Results */}
              {analysisResult && (
                <div className="results-section">
                  {analysisResult.success ? (
                    <>
                      <div className="results-header">
                        <span className="results-duration">
                          Completed in {analysisResult.duration.toFixed(1)}s
                        </span>
                        {analysisResult.thinking && (
                          <button
                            className="thinking-toggle"
                            onClick={() => setShowThinking(!showThinking)}
                          >
                            {showThinking ? "Hide" : "Show"} Thinking
                          </button>
                        )}
                      </div>
                      {showThinking && analysisResult.thinking && (
                        <div className="thinking-block">
                          <pre>{analysisResult.thinking}</pre>
                        </div>
                      )}
                      <div className="markdown-content">
                        <ReactMarkdown>{analysisResult.content}</ReactMarkdown>
                      </div>
                    </>
                  ) : (
                    <div className="error-block">
                      Error: {analysisResult.error}
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

export default App;
