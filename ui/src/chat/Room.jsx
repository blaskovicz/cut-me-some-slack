import React, { Component } from 'react';
// import PropTypes from 'prop-types';
import moment from 'moment';
import FontAwesome from 'react-fontawesome';
import Message from './Message';
import Api, { ApiListener } from '../lib/api';
import { getScroll } from '../lib/document';

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
      unread: null,
      startTs: moment(),
      messageTs: '',
      messages: [],
    };

    this.handleEnter = this.handleEnter.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.pushOutboundMessage = this.pushOutboundMessage.bind(this);
    this.viewUnreadMessages = this.viewUnreadMessages.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.handleConnectionStateChange = this.handleConnectionStateChange.bind(this);
    window.onscroll = this.onScroll.bind(this);

    Api.register(new (class RoomListener extends ApiListener {
      onMessage = msg => this.handleMessage(msg);
      onStateChange = (oldState, newState) => this.handleConnectionStateChange(oldState, newState);
    })());
  }
  componentDidMount() {
    const { slack: { channel } } = this.state;
    if (!channel) return;
    // console.log('[room.component-did-mount] requesting history for channel', channel);
    Api.historicalMessageRequest(channel.id);
  }
  componentDidUpdate(prevProps, prevState) {
    const prevMessageTs = prevState.messageTs;
    const prevChannel = prevState.slack.channel;
    const { messageTs, slack: { channel } } = this.state;
    // switched or loaded channels, backfill messages
    if ((channel && !prevChannel) || (channel && prevChannel && prevChannel.id !== channel.id)) {
      // console.log('[room.component-did-update] requesting history for channel', channel, ', was', prevChannel);
      Api.historicalMessageRequest(channel.id);
    }
    // if we didn't append a message to our list, don't scroll
    if (messageTs === prevMessageTs) {
      // console.log('[room.component-did-update] state changed but messageTs is the same');
      return;
    }
    const [, y] = getScroll();
    // if user is 85% or greater scrolled to the bottom, give them the new message...
    const scrollPerc = (100 / document.body.scrollHeight) * y;
    if (scrollPerc >= 85) {
      // console.log(`[room.component-did-update] scrolling to bottom (scrolled ${scrollPerc})`);
      window.scrollTo(0, document.body.scrollHeight);
    } else {
      // console.log(`[room.component-did-update] skipping scroll ${scrollPerc}`);
    }
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
    const { outboundMessage, slack: { channel } } = this.state;
    if (outboundMessage === '' || !channel) return;
    Api.sendMessage(outboundMessage, channel.id);
    window.scrollTo(0, document.body.scrollHeight);
    this.setState({ outboundMessage: '' });
  }

  onScroll() {
    const [, y] = getScroll();
    // if user is 85% or greater scrolled to the bottom, mark read.
    const scrollPerc = (100 / document.body.scrollHeight) * y;
    if (scrollPerc < 85) {
      return;
    }
    const { unread } = this.state;
    if (!unread) {
      return;
    }
    this.setState({ unread: null });
  }

  viewUnreadMessages() {
    this.setState({ unread: null });
    window.scrollTo(0, document.body.scrollHeight);
  }

  handleMessage(msg) {
    switch (msg.type) {
      case 'team-info': {
        let channel;
        const channels = {};
        msg.channels.forEach(c => {
          channels[c.id] = c;
          // TODO route param
          if (c.name === (process.env.REACT_APP_SLACK_CHANNEL || 'api-testing')) {
            channel = c;
          }
        });
        const users = {};
        msg.users.forEach(u => {
          users[u.id] = u;
        });
        this.setState({
          slack: {
            slack: msg.slack,
            username: msg.username,
            users,
            channel: this.state.slack.channel || channel,
            channels,
            emoji: msg.emoji,
          },
        });
        break;
      }
      case 'message': {
        const { messages, unread, startTs, slack: { channel } } = this.state;
        // TODO edits, emoji, deletes, sorting, etc

        // we got an invalid or old message, drop it.
        if (msg.ts == null) {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] dropping invalid message', msg);
          return;
        } else if (messages.length !== 0 && +(messages[messages.length - 1].ts) > +msg.ts) {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] dropping old message', msg);
          return;
        }

        const parsedMessageTs = moment(msg.ts * 1000);
        // message since we loaded the page, and in a different channel
        if (parsedMessageTs > startTs && channel && msg.channel.id !== channel.id) {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] dropping message in different channel', msg);
          return;
        }
        messages.push(msg);
        let newUnreads = unread;
        // if we got a message after our initial payload, it can be 'unread.'
        // make sure that it's not a duplicate.
        // if we're not scrolled to the bottom, append to unreads.
        // once we scroll to the bottom or click the message, mark as read.
        // otherwise, clear unreads.
        if (startTs < parsedMessageTs && (!unread || unread.since < parsedMessageTs)) {
          if (unread) {
            newUnreads.count += 1;
          } else {
            newUnreads = { count: 1, since: parsedMessageTs };
          }
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] adding unread message', msg, ', new unreads', newUnreads);
        } else {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] adding normal message', msg);

          // if we're still backfilling messages, auto-advance us
          // eslint-disable-next-line no-lonely-if
          if (parsedMessageTs < startTs) {
            window.scrollTo(0, document.body.scrollHeight);
          }
        }

        this.setState({
          messageTs: msg.ts,
          messages,
          unread: newUnreads,
        });
        break;
      }
      default: {
        // eslint-disable-next-line no-console
        console.warn('[room.handle-message] unhandled message', msg);
        break;
      }
    }
  }

  render() {
    const { handleChange, pushOutboundMessage, handleEnter, viewUnreadMessages } = this;
    const { unread, slack: { channel, slack, emoji, username, users, channels }, messages, outboundMessage, connectionState, connectionChangeTime } = this.state;
    return (
      <div>
        <div style={{ position: 'sticky', left: '0', top: '0', right: '0', zIndex: 1, background: '#fff' }} className="container">
          <div className="header" style={{ borderBottom: '1px solid #eee' }}>
            <h4 className="text-muted"><span id="header-slack">{slack}</span> Slack</h4>
            <h5 className="text-muted">
              <span id="header-username">@{username}</span>{' in '}
              <span id="header-channel" title={channel.id}>#{channel.name}</span>
            </h5>
            {unread &&
              <span
                onClick={viewUnreadMessages}
                style={{ cursor: 'pointer', width: '100%' }}
                className="badge badge-pill badge-primary"
              >
                {unread.count} New Messages Since {unread.since.format()}
                <FontAwesome style={{ marginLeft: '20px' }} name="times-circle-o" />
              </span>
            }
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
              style={{ display: 'inline-block', width: '80%', verticalAlign: 'top' }}
              onKeyPress={handleEnter}
              value={outboundMessage}
              onChange={handleChange}
              name="outboundMessage"
              type="text"
              className="form-control"
              id="message-text"
            />
            <button
              style={{ width: '20%', verticalAlign: 'top' }}
              disabled={!channel || outboundMessage === ''}
              id="message-submit"
              type="button"
              className={`btn btn-${(!channel || outboundMessage === '') ? 'secondary' : 'primary'}`}
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
