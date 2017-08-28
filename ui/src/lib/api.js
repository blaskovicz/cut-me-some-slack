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
    getState = () => this.sock ? this.sock.readyState : WebSocket.CLOSED;
    bindSock() {
        if (this.sock && this.sock.readyState !== WebSocket.CLOSED) this.sock.close();
        // TODO switch to non-native ws lib
        try {
            console.log(`[api.bind-sock] at=connecting uri=${WS_URI} backoff=${this.backoffCurrent}`)        
            this.sock =  new WebSocket(WS_URI);
            this.sock.onmessage = this.onMessage.bind(this);
            this.sock.onclose = this.onClose.bind(this);
            this.sock.onopen = this.onOpen.bind(this);
            this.sock.onerror = this.onError.bind(this);
        } catch(e) {
            console.error(e);
            this.stateChange(); // continue backoff
        }
    }
    register(listener){
        let foundListner = this.listeners.indexOf(listener);
        if (foundListner !== -1) return;
        this.listeners.push(listener);
    }
    unRegister(listener){
        let foundListner = this.listeners.indexOf(listener);
        if (foundListner === -1) return;
        this.listeners.splice(foundListner, 1);
    }
    stateChange(){
        const oldState = this.state;
        const newState = this.getState();
        if (oldState !== newState){
            console.log(`[api.state-change] from=${oldState} to=${newState} backoff=${this.backoffCurrent}`)
            this.state = newState;
            this.onStateChange(oldState, newState);
        } else {
            console.log(`[api.state-change] still=${newState} backoff=${this.backoffCurrent}`)
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
        this.listeners.forEach((l) => {
            let f = l.onStateChange;
            if (typeof f === 'function') f(oldState, newState);
        });       
    }
    onError(...args) {
        //this.stateChange();
        this.listeners.forEach((l) => {
            let f = l.onError;
            if (typeof f === 'function') f(...args);
        });
    }
    onOpen(...args) {
        this.stateChange();
        this.listeners.forEach((l) => {
            let f = l.onOpen;
            if (typeof f === 'function') f(...args);
        });
    }
    onMessage(e) {
        if (typeof e.data !== 'string') return;
        e.data.split('\n').forEach((rawMessage) => {
            if (rawMessage === '') return;
            let msg = JSON.parse(rawMessage);
            this.listeners.forEach((l) => {
                let f = l.onMessage;
                if (typeof f === 'function') f(msg);
            });
        });
    }
    onClose(...args) {
        this.stateChange();
        this.listeners.forEach((l) => {
            let f = l.onClose;
            if (typeof f === 'function') f(...args);
        });
    }
    sendMessage(text) {
        this.sock.send(JSON.stringify({ text, type: 'message' }));
    }
}
export class ApiListener {
    onOpen(...args){}
    onError(...args){}
    onClose(...args){}
    onMessage(msg){}
    onStateChange(oldState, newState){}
}
export default new Api();

