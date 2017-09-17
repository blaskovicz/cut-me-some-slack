import React, { Component } from 'react';
import PropTypes from 'prop-types';
import moment from 'moment';
import FontAwesome from 'react-fontawesome';
import { withRouter } from 'react-router-dom';
import Message from './Message';
import Api, { ApiListener } from '../lib/api';
import { getScroll } from '../lib/document';
import './Room.css';

export default withRouter(class Room extends Component {

  static propTypes = {
    match: PropTypes.shape({
      params: PropTypes.shape({
        channelID: PropTypes.string.isRequired,
      }).isRequired,
    }).isRequired,
    history: PropTypes.object.isRequired,
  };
  static freshState() {
    return {
      connectionChangeTime: null,
      connectionState: null,
      outboundMessage: '',
      switchChannelText: '',
      slack: {
        channel: null,
        slack: '', // team
        icon: '',
        user: null,
        users: {},
        channels: {},
        emoji: {},
      },
      unread: null,
      startTs: moment(),
      messageTs: '',
      messages: [],
      switchingChannels: false,
    };
  }
  constructor(props) {
    super(props);
    this.state = this.constructor.freshState();
    this.handleEnter = this.handleEnter.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.pushOutboundMessage = this.pushOutboundMessage.bind(this);
    this.viewUnreadMessages = this.viewUnreadMessages.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.handleConnectionStateChange = this.handleConnectionStateChange.bind(this);
    this.toggleSwitchChannels = this.toggleSwitchChannels.bind(this);
    this.changeChannel = this.changeChannel.bind(this);
    this.filterSwitchChannels = this.filterSwitchChannels.bind(this);
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
  componentWillReceiveProps(nextProps) {
    if (this.props.match.params.channelID !== nextProps.match.params.channelID) {
      const freshState = this.constructor.freshState();
      // copy certain state keys if we have them since we may not get them again
      if (this.state.slack) {
        freshState.slack = this.state.slack;
      }
      freshState.startTs = this.state.startTs;
      this.setState(freshState);
    }
  }
  componentDidUpdate(prevProps, prevState) {
    const prevMessageTs = prevState.messageTs;
    const prevChannel = prevState.slack.channel;
    const { messageTs, slack: { channel } } = this.state;
    // switched or loaded channels, backfill messages
    if (
      (channel && !prevChannel) ||
      (channel && prevChannel && prevChannel.id !== channel.id) ||
      prevProps.match.params.channelID !== this.props.match.params.channelID
    ) {
      // console.log('[room.component-did-update] requesting history for channel', channel, ', was', prevChannel);
      Api.historicalMessageRequest(this.props.match.params.channelID);
    }
    // if we didn't append a message to our list, don't scroll
    if (messageTs === prevMessageTs) {
      // console.log('[room.component-did-update] state changed but messageTs is the same');
      return;
    }
    // if user is past the threshold, give them the new message...
    if (this.constructor.pastScrollThreshold()) {
      // console.log(`[room.component-did-update] scrolling to bottom (scrolled ${scrollPerc})`);
      this.constructor.scrollToBottom();
    } else {
      // console.log(`[room.component-did-update] skipping scroll ${scrollPerc}`);
    }
  }

  static scrollToBottom() {
    // eslint-disable-next-line no-console
    console.log('[room.scroll-to-bottom]');
    window.scrollTo(0, document.body.scrollHeight);
  }
  static pastScrollThreshold() {
    const [, y] = getScroll();
    const scrollPerc = (100 / document.body.scrollHeight) * y;
    return scrollPerc >= 85; // 85% down the page
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
    this.constructor.scrollToBottom();
    this.setState({ outboundMessage: '' });
  }

  onScroll() {
    // if user is past scroll threshold, mark read.
    if (!this.constructor.pastScrollThreshold()) {
      return;
    }
    const { unread } = this.state;
    if (!unread) {
      return;
    }
    this.setState({ unread: null });
  }

  toggleSwitchChannels() {
    this.setState({ switchingChannels: !this.state.switchingChannels, switchChannelText: '' });
  }
  changeChannel(newChannel) {
    const { match: { params: { channelID } }, history } = this.props;
    if (newChannel.id === channelID) return;
    // eslint-disable-next-line no-console
    console.log(`[room.change-channel] changing channel to ${newChannel.id} per user request.`);
    history.push(`/messages/${newChannel.id}`);
    const { slack } = this.state;
    slack.channel = newChannel;
    this.setState({ slack });
  }
  filterSwitchChannels(e) {
    this.setState({ switchChannelText: e.target.value.toLowerCase() || '' });
  }

  viewUnreadMessages() {
    this.setState({ unread: null });
    this.constructor.scrollToBottom();
  }

  handleMessage(msg) {
    switch (msg.type) {
      case 'auth': {
        const { slack } = this.state;
        slack.user = msg.user;
        this.setState({ slack });
        break;
      }
      case 'team-info': {
        let channel;
        const channels = {};
        const { match: { params: { channelID } }, history } = this.props;
        msg.channels.forEach(c => {
          channels[c.id] = c;
          if (c.name === channelID) {
            // eslint-disable-next-line no-console
            console.log(`[room.handle-message] user requested #${c.name}; redirecting to id ${c.id}`);
            history.push(`/messages/${c.id}`);
            channel = c;
          } else if (c.id === channelID) {
            channel = c;
          }
        });

        if (!channel) {
          channel = msg.channels[0];
          // TODO grab general channel
          // eslint-disable-next-line no-console
          console.log(`[room.handle-message] user requested #${channelID} but it wasn't found; redirecting to id ${channel.id}`);
          history.push(`/messages/${channel.id}`);
        }

        const users = {};
        msg.users.forEach(u => {
          users[u.id] = u;
        });
        this.setState({
          slack: {
            slack: msg.slack,
            icon: msg.icon,
            users,
            channel,
            channels,
            emoji: msg.emoji,
          },
        });
        break;
      }
      case 'message': {
        // TODO there's an issue here with missing dropped messages on reconnect
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
        let foundMessageTs = false;
        // eslint-disable-next-line consistent-return
        messages.forEach(m => {
          if (m.ts === msg.ts) {
            foundMessageTs = true;
            return false;
          }
        });
        if (foundMessageTs) {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] dropping duplicate message', msg);
          return;
        }
        messages.push(msg); // TODO better handle out-of-order messages
        let newUnreads = unread;
        // if we got a message after our initial payload, it can be 'unread.'
        // make sure that it's not a duplicate.
        // if we're not scrolled to the bottom, append to unreads.
        // once we scroll to the bottom or click the message, mark as read.
        // otherwise, clear unreads.
        if (
          startTs < parsedMessageTs &&
          (!unread || unread.since < parsedMessageTs) &&
          !this.constructor.pastScrollThreshold()
        ) {
          if (unread) {
            newUnreads.count += 1;
          } else {
            newUnreads = { count: 1, since: parsedMessageTs };
          }
          // eslint-disable-next-line no-console
          console.log(
            '[room.handle-message] adding unread message', msg,
            ', new unreads', newUnreads,
            ', startTs=', startTs.format(),
            ', parsedMessageTs=', parsedMessageTs.format(),
          );
        } else {
          // eslint-disable-next-line no-console
          // console.log('[room.handle-message] adding normal message', msg);

          // if we're still backfilling messages, auto-advance us
          // eslint-disable-next-line no-lonely-if
          if (parsedMessageTs < startTs) {
            this.constructor.scrollToBottom();
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
    // TODO show loading while waiting for team info, messages, etc
    const { handleChange, pushOutboundMessage, handleEnter, viewUnreadMessages, toggleSwitchChannels, filterSwitchChannels, changeChannel } = this;
    const { switchChannelText, switchingChannels, unread, slack: { channel, icon, slack, emoji, user, users, channels }, messages, outboundMessage, connectionState, connectionChangeTime } = this.state;
    return (
      <div style={{ background: '#303E4D' }}>
        <div style={{ position: 'sticky', left: '0', top: '0', right: '0', zIndex: 1, background: '#303E4D' }} className="container">
          <div className="header" style={{ color: '#fff', borderBottom: '1px solid #000', paddingTop: '5px', paddingBottom: '5px' }}>
            <div style={{ width: '64px', display: 'inline-block', verticalAlign: 'top', marginRight: '5px' }}>
              {icon &&
                <img
                  src={icon}
                  alt="team icon"
                  style={{
                    width: '64px',
                    height: '64px',
                    borderRadius: '5px',
                  }}
                  className="card-image"
                />
              }
            </div>
            <div style={{ width: '75%', display: 'inline-block', verticalAlign: 'top' }}>
              <h4>
                {slack && <span>{slack}</span>}
                {slack && ' Slack'}
              </h4>
              <h5 >
                {user && <span style={{ opacity: '0.6' }}>@{user.username}</span>}
                {user && channel && <span style={{ opacity: '0.6' }}> in </span>}
                {channel && (
                  switchingChannels ?
                    <div style={{ display: 'inline-block' }}>
                      <input onChange={filterSwitchChannels} value={switchChannelText} style={{ border: '0', borderRadius: '5px' }} />
                    </div> :
                    <span
                      style={{ opacity: '0.6' }}
                      className="channel-switcher"
                      title={channel.id}
                      onClick={toggleSwitchChannels}
                    >
                      #{channel.name}
                    </span>
                )
                }
              </h5>
              {switchingChannels &&
                <div
                  className="channel-picker"
                  style={{}}
                  onMouseLeave={toggleSwitchChannels}
                >
                  {Object.keys(channels).map(k => {
                    const channelName = channels[k].name;
                    if (filterSwitchChannels !== '' && channelName.toLowerCase().indexOf(switchChannelText) === -1) {
                      // eslint-disable-next-line array-callback-return
                      return;
                    }
                    // eslint-disable-next-line consistent-return
                    return (
                      <div
                        className={`channel-picker-row ${k === channel.id ? 'current-channel' : ''}`}
                        key={k}
                        onClick={() => changeChannel(channels[k])}
                      >
                        #{channelName}
                      </div>
                    );
                  })}
                </div>
              }
              {unread &&
                <span
                  onClick={viewUnreadMessages}
                  style={{ cursor: 'pointer', width: '100%', display: 'block' }}
                  className="badge badge-pill badge-primary"
                >
                  {unread.count} New Messages Since {unread.since.format('MMM Do, h:mm a')}
                  <FontAwesome style={{ marginLeft: '20px' }} name="times-circle-o" />
                </span>
              }
              {connectionState !== null && connectionState !== WebSocket.OPEN &&
                <span
                  className="badge badge-pill badge-warning"
                  style={{ width: '100%', display: 'block' }}
                >
                  <i>Re-establishing connection to Slack (since {connectionChangeTime.format('MMM Do, h:mm a')}).</i>
                </span>
              }
            </div>
          </div>
        </div>
        <div style={{ paddingBottom: '40px', paddingTop: '20px' }} className="container">
          <div className="messages" style={{ marginBottom: '20px' }}>
            {messages.map(msg =>
              <Message emoji={emoji} key={msg.ts} users={users} channels={channels} msg={msg} />)}
          </div>
        </div>
        <div style={{ height: '50px', position: 'fixed', left: '0', bottom: '0', right: '0', zIndex: 1, background: '#303E4D', paddingTop: '5px', paddingBottom: '10px' }} className="container">
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
});
