import React, { Component } from 'react';
import ReactMarkdown from 'react-markdown';
import emojione from 'emojione';
import PropTypes from 'prop-types';
import moment from 'moment';

export default class Message extends Component {
  static propTypes = {
    users: PropTypes.object,
    channels: PropTypes.object,
    emoji: PropTypes.object,
    msg: PropTypes.shape({
      text: PropTypes.string,
      ts: PropTypes.string.isRequired,
      user: PropTypes.shape({
        avatar_url: PropTypes.string,
        username: PropTypes.string.isRequired,
      }),
    }).isRequired,
  };

  parsedMessage() {
    const { msg, users, channels, emoji } = this.props;
    if (!msg.text || msg.parsed) return msg;
    // correct anything out of the norm in the text.
    let startIndex = -1;
    let endIndex = -1;
    let labelIndex = -1;
    let emojiIndex = -1;
    for (let i = 0; i < msg.text.length; i++) {
      const charAt = msg.text[i];
      if (charAt === '<') {
        startIndex = i;
        endIndex = -1;
        labelIndex = -1;
        emojiIndex = -1;
      } else if (charAt === '|') {
        labelIndex = i;
        emojiIndex = -1;
      } else if (charAt === '>' && startIndex > -1) {
        endIndex = i;
        emojiIndex = -1;
        // eg: <#c1234|misc> -> #c1234
        const token = msg.text.substring(
          startIndex + 1,
          labelIndex === -1 ? endIndex : labelIndex,
        );
        let newToken;
        if (token[0] === '#') {
          // channel
          const matchedChannel = channels && channels[token.substring(1)];
          if (matchedChannel) {
            newToken = `#${matchedChannel.name}`;
          }
        } else if (token[0] === '@') {
          // username
          const matchedUser = users && users[token.substring(1)];
          if (matchedUser) {
            newToken = `@${matchedUser.username}`;
          }
        } else {
          // link
          newToken = labelIndex !== -1 ?
            `[${msg.text.substring(labelIndex + 1, endIndex)}](${token})` : `[${token}](${token})`;
        }
        if (newToken && token !== newToken) {
          // console.log(`[message.parse] replacing ${msg.text.substring(startIndex, endIndex + 1)} (s=${startIndex}, e=${endIndex}, l=${labelIndex}) with ${newToken}`);
          msg.text = msg.text.slice(0, startIndex) + newToken + msg.text.slice(endIndex + 1);
        }
        startIndex = -1;
        endIndex = -1;
        labelIndex = -1;
      } else if (charAt === ':') {
        // fix random emojis that don't proc because #emojioneBugs
        if (emojiIndex === -1) {
          emojiIndex = i;
        } else {
          const token = msg.text.substring(emojiIndex + 1, i);
          let newToken;
          // console.log(`[message.parse] emoji ${emoji}`);
          if (token === '+1') {
            newToken = ':thumbsup:';
          } else {
            let targetEmoji = emoji[token];
            // resolve aliases eg: foo -> alias:bar -> bar -> alias:baz -> baz -> baz/url.png
            while (targetEmoji !== undefined && targetEmoji.indexOf('alias:') === 0) {
              targetEmoji = emoji[targetEmoji.split('alias:')[1]];
            }
            if (targetEmoji) {
              newToken = `<img alt="${token}" title="${token}" src="${targetEmoji}" style="width: 32px; height: 32px">`;
            }
          }

          if (newToken && token !== newToken) {
            msg.text = msg.text.slice(0, emojiIndex) + newToken + msg.text.slice(i + 1);
          }
          emojiIndex = -1;
        }
      } else if (!/[a-zA-Z0-9_\-+]/.test(charAt)) {
        emojiIndex = -1;
      }
    }
    // TODO mix in custom emojis (https://api.slack.com/methods/emoji.list)
    msg.text = emojione.toImage(msg.text);
    msg.parsed = true;
    return msg;
  }
  render() {
    const msg = this.parsedMessage();
    const shortTime = moment(msg.ts * 1000).format('MMM Do, h:mm a');
    const longTime = moment(msg.ts * 1000).format();
    return (
      <div key={msg.ts} className="card" style={{ marginTop: '5px' }}>
        <div className="card-block" style={{ padding: '.5rem' }}>
          <div style={{ display: 'inline-block', width: '80px', verticalAlign: 'top' }}>
            {msg.user && msg.user.avatar_url &&
              <img
                src={msg.user.avatar_url}
                alt="avatar"
                style={{
                  width: '80px',
                  height: '80px',
                  marginRight: '5px',
                  borderRadius: '5px',
                }}
                className="card-image"
              />
            }
          </div>
          <div style={{ display: 'inline-block', width: '75%', verticalAlign: 'top', marginLeft: '10px' }}>
            <h6 className="card-title">
              {msg.user && <span style={{ marginRight: '5px', fontWeight: 'bold' }}>{msg.user.username}</span>}
              <span style={{ fontSize: '10pt', color: '#929191' }} title={longTime}>{shortTime}</span>
            </h6>
            {/* TODO links, sigils and whatnot */}
            <div style={{ whiteSpace: 'pre-wrap', overflow: 'auto' }}>
              {msg.text && <ReactMarkdown source={msg.text} />}
            </div>
            {/* <!--a href="#" className="card-link">Card link</a-->
            <!--a href="#" className="card-link">Another link</a-->*/}
          </div>
        </div>
      </div>
    );
  }
}
