let chatHistory = [];

document.addEventListener('DOMContentLoaded', () => {
    const sendBtn = document.getElementById('sendBtn');
    const messageInput = document.getElementById('messageInput');
    const imagePrompt = document.getElementById('imagePrompt');
    const generateImage = document.getElementById('generateImage');
    const newChat = document.getElementById('newChat');
    const themeToggle = document.getElementById('themeToggle');
    const chatContainer = document.getElementById('chatContainer');

    // Send message
    sendBtn.addEventListener('click', sendMessage);
    messageInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    // Generate image
    generateImage.addEventListener('click', generateImageFunc);

    // New chat
    newChat.addEventListener('click', () => {
        fetch('/new-chat', { method: 'POST' });
        chatHistory = [];
        chatContainer.innerHTML = '';
    });

    // Theme toggle
    themeToggle.addEventListener('change', (e) => {
        document.documentElement.setAttribute('data-theme', e.target.checked ? 'dark' : 'light');
    });

    function sendMessage() {
        const message = messageInput.value.trim();
        if (!message) return;

        addMessage('user', message);
        messageInput.value = '';

        fetch('/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ messages: [{ role: 'user', content: message }] })
        })
        .then(res => res.json())
        .then(data => addMessage('ai', data.content));
    }

    function generateImageFunc() {
        const prompt = imagePrompt.value.trim();
        if (!prompt) return;

        addMessage('user', `/imagine ${prompt}`);
        imagePrompt.value = '';

        fetch('/image', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt })
        })
        .then(res => res.blob())
        .then(blob => {
            const url = URL.createObjectURL(blob);
            const img = document.createElement('img');
            img.src = url;
            img.className = 'image-preview';
            chatContainer.appendChild(img);
        });
    }

    function addMessage(role, content) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        messageDiv.innerHTML = `
            <div class="avatar ${role}">${role === 'user' ? '👤' : '✨'}</div>
            <div>
                <div class="bubble">${content}</div>
                <div class="message-actions">
                    <button class="action-btn" onclick="copyMessage(this)">📋 Copy</button>
                    <button class="action-btn" onclick="rewriteMessage(this)">✏️ Rewrite</button>
                </div>
            </div>
        `;
        
        chatContainer.appendChild(messageDiv);
        chatContainer.scrollTop = chatContainer.scrollHeight;
    }

    window.copyMessage = (btn) => {
        navigator.clipboard.writeText(btn.parentElement.previousElementSibling.textContent);
        btn.textContent = '✅ Copied!';
        setTimeout(() => btn.textContent = '📋 Copy', 2000);
    };

    window.rewriteMessage = (btn) => {
        const message = btn.parentElement.previousElementSibling.textContent;
        fetch('/rewrite', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message })
        })
        .then(res => res.json())
        .then(data => {
            btn.closest('.message').querySelector('.bubble').textContent = data.rewritten;
        });
    };
});
