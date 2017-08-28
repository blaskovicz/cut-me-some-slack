import React, { Component } from 'react';
//import PropTypes from 'prop-types';
import Api, { ApiListener } from '../lib/api';
import moment from 'moment';

export default class Room extends Component {

    constructor(props) {
        super(props);

        this.state = {
            connectionChangeTime: null,
            connectionState: null,
            outboundMessage: '',
            slack: {
                channel: '',
                slack: '', // team
                username: '',
            },
            messages: [],
        }

        this.handleEnter = this.handleEnter.bind(this);
        this.handleChange = this.handleChange.bind(this);
        this.pushOutboundMessage = this.pushOutboundMessage.bind(this);
        this.handleMessage = this.handleMessage.bind(this);
        this.handleConnectionStateChange = this.handleConnectionStateChange.bind(this);

        Api.register(new (class RoomListener extends ApiListener {
            onMessage = (msg) => this.handleMessage(msg);
            onStateChange = (oldState, newState) => this.handleConnectionStateChange(oldState, newState);
        })());
    }

    handleConnectionStateChange(oldState, newState) {
        if (newState === WebSocket.OPEN) {
            this.setState({ connectionState: newState, connectionChangeTime: null });
        } else {
            let { connectionChangeTime } = this.state;
            if (connectionChangeTime === null) connectionChangeTime = moment(); // only allocate a time at the oldest drop
            this.setState({ connectionState: newState, connectionChangeTime });
        }
        
    }
    handleEnter(e) {
        if (e.key !== 'Enter') return;
        this.pushOutboundMessage();
    }
    handleChange(e) {
        this.setState({[e.target.name]: e.target.value});
    }
    pushOutboundMessage() {
        const { outboundMessage } = this.state;
        if (outboundMessage === '') return;
        Api.sendMessage(outboundMessage);
        this.setState({outboundMessage: ''});
    }

    handleMessage(msg) {
        switch(msg.type) {
            case 'team-info':
                this.setState({
                    slack: {
                        channel: msg.channel,
                        slack: msg.slack,
                        username: msg.username,
                    }
                });
                break;
            case 'message':
                const { messages } = this.state;
                // TODO edits, emoji, deletes, sorting, etc

                // we got an invalid or old message, drop it.
                if (!msg.ts) return;
                if (messages.length !== 0 && +(messages[messages.length-1].ts) > +msg.ts){
                    console.log("Dropping old message", msg);
                    return;
                }
                messages.push(msg);
                this.setState({
                    messages,
                });
                break;
            default:
                console.log("Unhandled message", msg);
                break;
        }
    }

    render() {
        const { handleChange, pushOutboundMessage, handleEnter } = this;
        const { slack : { channel, slack, username }, messages, outboundMessage, connectionState, connectionChangeTime } = this.state;
        return (
            <div className="container">
                <div className="header" style={{borderBottom: '1px solid #eee'}}>
                    <h3 className="text-muted"><span id='header-slack'>{slack}</span> Slack <small></small></h3>
                    <h4 className="text-muted"><span id='header-channel'>{channel}</span></h4>
                    <h4 className="text-muted"><span id='header-username'>{username}</span></h4>
                </div>
                <div className="messages" style={{marginBottom: '20px'}}>
                    {messages.map((msg) => 
                        <div key={msg.ts} className="card" style={{marginTop: '5px'}}>
                            <div className="card-block" style={{padding: '.5rem'}}>
                                {msg.user && msg.user.avatar_url ?
                                    <div style={{float: 'left'}}>
                                        <img src={msg.user.avatar_url} alt='avatar' style={{width: '80px', height: '80px', marginRight: '5px'}} className='card-image'/>
                                    </div> : ''
                                }
                                <div style={{float: 'left'}}>
                                    <h4 className="card-title">{msg.user ? msg.user.username : ''}</h4>
                                    <h6 className="card-subtitle mb-2 text-muted">
                                        {moment(msg.ts * 1000).format()}
                                    </h6>
                                    {/* TODO links, sigils and whatnot */}
                                    <p className="card-text" style={{overflow: 'auto'}}>{msg.text || ''}</p>
                                    {/*<!--a href="#" className="card-link">Card link</a-->
                                    <!--a href="#" className="card-link">Another link</a-->*/}
                                </div>
                            </div>
                        </div>
                    )}
                    {(connectionState !== null && connectionState !== WebSocket.OPEN) ?
                        <div className="alert alert-warning" role="alert">
                            <i>Re-establishing connection to Slack (since {connectionChangeTime.format()}).</i>
                        </div> : ''
                    }
                </div>
                <div id='message-new-controls'>
                    <div className='form-group row'>
                        <div className='col-11'>
                            <input onKeyPress={handleEnter} value={outboundMessage} onChange={handleChange} name='outboundMessage' type='text' className='form-control' id='message-text'/>
                            </div>
                            <div className='col-1'>
                            <button disabled={outboundMessage === ''} id='message-submit' type='button' className='btn btn-primary' onClick={pushOutboundMessage}>Send</button>
                        </div>
                    </div>
                </div>
            </div>
        );
    }
}