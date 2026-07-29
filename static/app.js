const form = document.querySelector('#chat-form');
const input = document.querySelector('#message-input');
const messages = document.querySelector('#messages');

function addMessage(text, kind) {
  const el = document.createElement('div');
  el.className = `message ${kind}`;
  el.textContent = text;
  messages.appendChild(el);
  messages.scrollTop = messages.scrollHeight;
  return el;
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  const message = input.value.trim();
  if (!message) return;
  addMessage(message, 'user');
  input.value = '';
  input.disabled = true;
  const pending = addMessage('Llama is thinking…', 'bot pending');
  try {
    const response = await fetch('/api/chat', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({message}) });
    const data = await response.json();
    pending.textContent = data.reply || 'Something went wrong. Please try again.';
  } catch {
    pending.textContent = 'The server is unavailable. Start the Go application and try again.';
  } finally {
    pending.classList.remove('pending');
    input.disabled = false;
    input.focus();
  }
});
