import React, { Component } from 'react';
// import PropTypes from 'prop-types';
import moment from 'moment';
import Message from './Message';
import Api, { ApiListener } from '../lib/api';

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
        users: {},
        channels: {},
        emoji: {},
      },
      messages: [],
    };

    this.handleEnter = this.handleEnter.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.pushOutboundMessage = this.pushOutboundMessage.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.handleConnectionStateChange = this.handleConnectionStateChange.bind(this);

    Api.register(new (class RoomListener extends ApiListener {
      onMessage = msg => this.handleMessage(msg);
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
    this.setState({ [e.target.name]: e.target.value });
  }
  pushOutboundMessage() {
    const { outboundMessage } = this.state;
    if (outboundMessage === '') return;
    Api.sendMessage(outboundMessage);
    this.setState({ outboundMessage: '' });
  }

  handleMessage(msg) {
    switch (msg.type) {
      case 'team-info': {
        const channels = {};
        msg.channels.forEach(c => {
          channels[c.id] = c;
        });
        const users = {};
        msg.users.forEach(u => {
          users[u.id] = u;
        });
        this.setState({
          slack: {
            channel: msg.channel,
            slack: msg.slack,
            username: msg.username,
            users,
            channels,
            emoji: msg.emoji,
          },
        });
        break;
      }
      case 'message': {
        const { messages } = this.state;
        // TODO edits, emoji, deletes, sorting, etc

        // we got an invalid or old message, drop it.
        if (msg.ts == null) {
          // eslint-disable-next-line no-console
          console.log('Dropping invalid message', msg);
          return;
        } else if (messages.length !== 0 && +(messages[messages.length - 1].ts) > +msg.ts) {
          // eslint-disable-next-line no-console
          console.log('Dropping old message', msg);
          return;
        }
        messages.push(msg);
        this.setState({
          messages,
        });
        break;
      }
      default: {
        // eslint-disable-next-line no-console
        console.log('Unhandled message', msg);
        break;
      }
    }
  }

  render() {
    const { handleChange, pushOutboundMessage, handleEnter } = this;
    const { slack: { channel, slack, emoji, username, users, channels }, messages, outboundMessage, connectionState, connectionChangeTime } = this.state;
    return (
      <div>
        <div style={{ position: 'sticky', left: '0', top: '0', right: '0', zIndex: 1, background: '#fff' }} className="container">
          <div className="header" style={{ borderBottom: '1px solid #eee' }}>
            <h4 className="text-muted"><span id="header-slack">{slack}</span> Slack</h4>
            <h5 className="text-muted">
              <span id="header-username">@{username}</span>{' in '}
              <span id="header-channel">#{channel}</span>
            </h5>
          </div>
        </div>
        <div style={{ paddingBottom: '40px', paddingTop: '40px' }} className="container">
          <div className="messages" style={{ marginBottom: '20px' }}>
            {messages.map(msg =>
              <Message emoji={emoji} key={msg.ts} users={users} channels={channels} msg={msg} />)}
            {(connectionState !== null && connectionState !== WebSocket.OPEN) ?
              <div className="alert alert-warning" role="alert">
                <i>Re-establishing connection to Slack (since {connectionChangeTime.format()}).</i>
              </div> : ''
            }
          </div>
        </div>
        <div style={{ height: '50px', position: 'fixed', left: '0', bottom: '0', right: '0', zIndex: 1, background: '#fff' }} className="container">
          <div id="message-new-controls">
            <input
              style={{ display: 'inline-block', width: '90%' }}
              onKeyPress={handleEnter}
              value={outboundMessage}
              onChange={handleChange}
              name="outboundMessage"
              type="text"
              className="form-control"
              id="message-text"
            />
            <button
              style={{ width: '10%' }}
              disabled={outboundMessage === ''}
              id="message-submit"
              type="button"
              className="btn btn-primary"
              onClick={pushOutboundMessage}
            >
              Send
            </button>
          </div>
        </div>
      </div >
    );
  }
}
