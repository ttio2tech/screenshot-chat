export interface ScreenshotResult {
  filePath: string;
  fileName: string;
  success: boolean;
  error?: string;
}

export interface ScreenshotInfo {
  fileName: string;
  filePath: string;
  size: number;
  modTime: string;
}

export interface AnalysisResult {
  content: string;
  thinking?: string;
  success: boolean;
  error?: string;
  duration: number;
}

export interface OllamaStatus {
  available: boolean;
  modelLoaded: boolean;
  error?: string;
}
