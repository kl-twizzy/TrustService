chrome.runtime?.onInstalled?.addListener(() => {
  chrome.storage.local.set({
    apiBaseUrl: "http://localhost:8080"
  });
});
