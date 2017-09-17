import jwtDecode from 'jwt-decode';
import { WS_URI } from './env';

// TODO wrapper around events and reg/unreg
class Api {
  constructor() {
    this.backoffInterval = 1000;
    this.backoffCurrent = 0;
    this.listeners = [];
    this.state = WebSocket.CLOSED;
    this.stateChange = this.stateChange.bind(this);
    this.bindSock = this.bindSock.bind(this);
    this.bindSock();
  }
  // WebSocket.CLOSING,CLOSED,CONNECTING,OPEN
  // eslint-disable-next-line no-confusing-arrow
  getState = () => this.sock ? this.sock.readyState : WebSocket.CLOSED;
  bindSock() {
    if (this.sock && this.sock.readyState !== WebSocket.CLOSED) this.sock.close();
    // TODO switch to non-native ws lib
    try {
      // eslint-disable-next-line no-console
      console.log(`[api.bind-sock] at=connecting uri=${WS_URI} backoff=${this.backoffCurrent}`);
      this.sock = new WebSocket(WS_URI);
      this.sock.onmessage = this.onMessage.bind(this);
      this.sock.onclose = this.onClose.bind(this);
      this.sock.onopen = this.onOpen.bind(this);
      this.sock.onerror = this.onError.bind(this);
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error(e);
      this.stateChange(); // continue backoff
    }
  }
  register(listener) {
    const foundListener = this.listeners.indexOf(listener);
    if (foundListener !== -1) return;
    this.listeners.push(listener);
  }
  unRegister(listener) {
    const foundListener = this.listeners.indexOf(listener);
    if (foundListener === -1) return;
    this.listeners.splice(foundListener, 1);
  }
  stateChange() {
    const oldState = this.state;
    const newState = this.getState();
    if (oldState !== newState) {
      // eslint-disable-next-line no-console
      console.log(`[api.state-change] from=${oldState} to=${newState} backoff=${this.backoffCurrent}`);
      this.state = newState;
      this.onStateChange(oldState, newState);
    } else {
      // eslint-disable-next-line no-console
      console.log(`[api.state-change] still=${newState} backoff=${this.backoffCurrent}`);
    }

    if (newState === WebSocket.OPEN) {
      this.backoffCurrent = 0;
    } else if (newState === WebSocket.CLOSED) {
      // note this flaps between CONNECTING to CLOSED during backoff
      this.backoffCurrent += this.backoffInterval;
      setTimeout(() => this.bindSock(), this.backoffCurrent); // re-establish connection
    }
  }
  onStateChange(oldState, newState) {
    this.listeners.forEach(l => {
      const f = l.onStateChange;
      if (typeof f === 'function') f(oldState, newState);
    });
  }
  onError(...args) {
    // this.stateChange();
    this.listeners.forEach(l => {
      const f = l.onError;
      if (typeof f === 'function') f(...args);
    });
  }
  onOpen(...args) {
    this.stateChange();
    this.sendAuthMessage();
    this.listeners.forEach(l => {
      const f = l.onOpen;
      if (typeof f === 'function') f(...args);
    });
  }
  onMessage(e) {
    if (typeof e.data !== 'string') return;
    e.data.split('\n').forEach(rawMessage => {
      if (rawMessage === '') return;
      let msg = JSON.parse(rawMessage);
      // if it's an auth response, handle that here
      if (msg.type === 'auth') {
        const jwt = jwtDecode(msg.token);
        // eslint-disable-next-line no-console
        console.log(`[api.on-message] got auth response, now user ${jwt.user.username}`,
          (msg.warning ? `(warning ${msg.warning})` : ''),
        );
        localStorage.setItem('jwt', msg.token);

        // pre-process message
        msg = { type: 'auth', user: jwt.user };
      }

      // otherwise, let our attached listeners handle it
      this.listeners.forEach(l => {
        const f = l.onMessage;
        if (typeof f === 'function') f(msg);
      });
    });
  }
  onClose(...args) {
    this.stateChange();
    this.listeners.forEach(l => {
      const f = l.onClose;
      if (typeof f === 'function') f(...args);
    });
  }
  sendAuthMessage() {
    const token = localStorage.getItem('jwt');
    if (token) {
      const jwt = jwtDecode(token);
      if (jwt && jwt.user) {
        // eslint-disable-next-line no-console
        console.log(`[api.send-auth-message] requesting auth for user ${jwt.user.username}`);
      } else {
        // eslint-disable-next-line no-console
        console.log('[api.send-auth-message] requesting auth with malformed token', token, ', parsed as', jwt);
      }
    } else {
      // eslint-disable-next-line no-console
      console.log('[api.send-auth-message] sending anonymous auth request');
    }

    this.sock.send(JSON.stringify({ type: 'auth', token }));
  }
  sendMessage(text, channel) {
    this.sock.send(JSON.stringify({ text, type: 'message', channel_id: channel }));
  }
  historicalMessageRequest(channel) {
    this.sock.send(JSON.stringify({ type: 'history', channel_id: channel }));
  }
}

/* eslint-disable class-methods-use-this */
/* eslint-disable no-unused-vars */
export class ApiListener {
  onOpen(...args) { }
  onError(...args) { }
  onClose(...args) { }
  onMessage(msg) { }
  onStateChange(oldState, newState) { }
}
/* eslint-enable class-methods-use-this */
/* eslint-enable no-unused-vars */
export default new Api();

