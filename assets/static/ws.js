// WebSocket connection instance
let ws = null;
let messageHandlers = new Map();

function openWebSocket() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        return ws;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
    
    ws.onopen = function() {
        console.log('WebSocket connection established');
        if (typeof updateConnectionStatus === 'function') {
            updateConnectionStatus(true);
        }
    };
    
    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            // Call all registered handlers for this message type
            const handlers = messageHandlers.get(data.type) || [];
            handlers.forEach(handler => handler(data));
        } catch (e) {
            console.error('Error parsing message:', e);
            const errorHandlers = messageHandlers.get('error') || [];
            errorHandlers.forEach(handler => handler({ type: 'error', text: 'Error parsing message' }));
        }
    };
    
    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        if (typeof updateConnectionStatus === 'function') {
            updateConnectionStatus(false);
        }
    };
    
    ws.onclose = function() {
        console.log('WebSocket connection closed');
        if (typeof updateConnectionStatus === 'function') {
            updateConnectionStatus(false);
        }
        // Try to reconnect after 5 seconds
        setTimeout(openWebSocket, 5000);
    };
    
    return ws;
}

function addMessageHandler(type, handler) {
    if (!messageHandlers.has(type)) {
        messageHandlers.set(type, []);
    }
    messageHandlers.get(type).push(handler);
}

function removeMessageHandler(type, handler) {
    if (!messageHandlers.has(type)) return;
    const handlers = messageHandlers.get(type);
    const index = handlers.indexOf(handler);
    if (index !== -1) {
        handlers.splice(index, 1);
    }
}

function sendMessage(message) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        ws = openWebSocket();
    }
    
    // If the connection is still connecting, wait for it to open
    if (ws.readyState === WebSocket.CONNECTING) {
        ws.addEventListener('open', () => ws.send(JSON.stringify(message)));
    } else if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(message));
    }
}

// Initialize WebSocket connection when the script loads
openWebSocket();