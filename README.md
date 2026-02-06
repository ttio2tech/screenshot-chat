# README

## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.



● All phases complete. Here's a summary:                                                  
                                                                                          
  Implementation complete. All 4 phases executed successfully:                            
                                                                                          
  1. Phase 0 - Wails project initialized with react-ts template, ~/Screenshots directory  
  created                                                                                 
  2. Phase 1 - Go backend (app.go, main.go) with screenshot capture (scrot), Ollama vision
   API integration, file management, and base64 image serving                             
  3. Phase 2 - React/TypeScript frontend with dark theme, two-panel layout (gallery +
  preview/analysis), react-markdown rendering, thinking toggle, and elapsed time counter  
  4. Phase 3 - Production build successful (8.8MB binary)

  Notes:
  - Had to upgrade vite (3.x -> 7.x) and @vitejs/plugin-react (2.x -> 5.x) because
  react-markdown v10 uses Node.js subpath imports not supported by vite 3
  - Used -tags webkit2_41 build flag since Ubuntu 25.04 ships webkit2gtk-4.1 instead of
  4.0. For wails dev, you'll need: wails dev -tags webkit2_41

  To run:
  # Development mode:
  cd /home/ti/CascadeProjects/screenshotLocal_02_go
  wails dev -tags webkit2_41

  # Or run the built binary:
  ./build/bin/screenshotLocal_02_go




