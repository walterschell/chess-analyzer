function openWebSocket(onDataCB) {
    host = window.location.host;
    protocol = window.location.protocol === "https:" ? "wss" : "ws";
    ws = new WebSocket(`${protocol}://${host}/ws`);
    ws.onmessage = (event) => {
        onDataCB(event.data);
    };
    ws.onclose = (event) => {
        console.log("WebSocket closed.");
        onDataCB("WebSocket closed.");
    };
    ws.onerror = (event) => {
        onDataCB("WebSocket error.");
        console.error("WebSocket error:", event);
    };
    return ws;
}