export namespace main {
	
	export class AnalysisResult {
	    content: string;
	    thinking?: string;
	    success: boolean;
	    error?: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.thinking = source["thinking"];
	        this.success = source["success"];
	        this.error = source["error"];
	        this.duration = source["duration"];
	    }
	}
	export class ScreenshotInfo {
	    fileName: string;
	    filePath: string;
	    size: number;
	    modTime: string;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.filePath = source["filePath"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	    }
	}
	export class ScreenshotResult {
	    filePath: string;
	    fileName: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}

}

