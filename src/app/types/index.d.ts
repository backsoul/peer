export {};
declare global {
  interface Window {
    SpeechRecognition: any;  // this will be your variable name
    webkitSpeechRecognition: any;
    webkitAudioContext:any;
  }
}