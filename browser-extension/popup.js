const apiInput = document.getElementById("apiBaseUrl");
const currentUrlNode = document.getElementById("currentUrl");
const checkBtn = document.getElementById("checkBtn");
const statusNode = document.getElementById("status");
const resultCard = document.getElementById("resultCard");
const trustScoreNode = document.getElementById("trustScore");
const authScoreNode = document.getElementById("authScore");
const marketplaceNode = document.getElementById("marketplace");
const productTitleNode = document.getElementById("productTitle");
const sellerNameNode = document.getElementById("sellerName");
const recommendationNode = document.getElementById("recommendation");
const reasonsNode = document.getElementById("reasons");

let activeTabUrl = "";
let clientId = "";

init();

async function init() {
  const stored = await chrome.storage.local.get(["apiBaseUrl", "clientId"]);
  const { apiBaseUrl = "http://localhost:8080" } = stored;
  clientId = stored.clientId || `ext-${Math.random().toString(36).slice(2, 10)}`;
  await chrome.storage.local.set({ clientId });
  apiInput.value = apiBaseUrl;
  await refreshActiveTabUrl();
}

checkBtn.addEventListener("click", async () => {
  const apiBaseUrl = apiInput.value.trim() || "http://localhost:8080";
  await chrome.storage.local.set({ apiBaseUrl });

  await refreshActiveTabUrl();
  setLoading(true, "Отправляю ссылку на проверку...");

  try {
    if (!activeTabUrl || activeTabUrl.startsWith("chrome://")) {
      throw new Error("Открой страницу товара на OZON, WB или Яндекс Маркете.");
    }

    const response = await fetch(`${apiBaseUrl}/api/v1/trust/analyze-url`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Client-Id": clientId
      },
      body: JSON.stringify({
        product_url: activeTabUrl
      })
    });

    const rawText = await response.text();
    let data = null;

    try {
      data = JSON.parse(rawText);
    } catch {
      throw new Error("API вернул не JSON. Проверь, что в поле API URL указан http://localhost:8080");
    }

    if (!response.ok) {
      throw new Error(data.error || "Ошибка проверки");
    }

    renderResult(data);
    setLoading(false, "Проверка завершена.");
  } catch (error) {
    resultCard.classList.add("hidden");
    setLoading(false, error.message || "Не удалось проверить товар.");
  }
});

async function refreshActiveTabUrl() {
  setLoading(true, "Получаю ссылку из текущей вкладки...");

  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  activeTabUrl = tab?.url || "";
  currentUrlNode.textContent = activeTabUrl || "Не удалось определить URL текущей вкладки.";

  setLoading(false, "Можно запускать проверку.");
}

function setLoading(isLoading, message) {
  checkBtn.disabled = isLoading;
  checkBtn.textContent = isLoading ? "Проверяем..." : "Проверить товар";
  statusNode.textContent = message;
}

function renderResult(data) {
  resultCard.classList.remove("hidden");
  trustScoreNode.textContent = `${data.trust_score}/100`;
  authScoreNode.textContent = `${data.rating_authenticity}/100`;
  marketplaceNode.textContent = humanizeMarketplace(data.marketplace);
  productTitleNode.textContent = data.product.title;
  sellerNameNode.textContent = `Продавец: ${data.seller.name}`;
  recommendationNode.textContent = data.recommendation;

  reasonsNode.innerHTML = "";
  for (const reason of data.reasons) {
    const item = document.createElement("li");
    item.textContent = reason;
    reasonsNode.appendChild(item);
  }
}

function humanizeMarketplace(code) {
  if (code === "ozon") {
    return "Маркетплейс: OZON";
  }
  if (code === "wildberries") {
    return "Маркетплейс: Wildberries";
  }
  if (code === "yandex_market") {
    return "Маркетплейс: Яндекс Маркет";
  }
  return `Маркетплейс: ${code}`;
}
