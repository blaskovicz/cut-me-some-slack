import React, { Component } from 'react';
import PropTypes from 'prop-types';
import moment from 'moment';

export default class Message extends Component {
  static propTypes = {
    msg: PropTypes.shape({
      text: PropTypes.string,
      ts: PropTypes.string.isRequired,
      user: PropTypes.shape({
        avatar_url: PropTypes.string,
        username: PropTypes.string.isRequired,
      }),
    }).isRequired,
  };
  render() {
    const { msg } = this.props;
    return (
      <div key={msg.ts} className="card" style={{ marginTop: '5px' }}>
        <div className="card-block" style={{ padding: '.5rem' }}>
          {msg.user && msg.user.avatar_url ?
            <div style={{ float: 'left' }}>
              <img src={msg.user.avatar_url} alt="avatar" style={{ width: '80px', height: '80px', marginRight: '5px' }} className="card-image" />
            </div> : ''
          }
          <div style={{ float: 'left' }}>
            <h4 className="card-title">{msg.user ? msg.user.username : ''}</h4>
            <h6 className="card-subtitle mb-2 text-muted">
              {moment(msg.ts * 1000).format()}
            </h6>
            {/* TODO links, sigils and whatnot */}
            <p className="card-text" style={{ overflow: 'auto' }}>{msg.text || ''}</p>
            {/* <!--a href="#" className="card-link">Card link</a-->
            <!--a href="#" className="card-link">Another link</a-->*/}
          </div>
        </div>
      </div>
    );
  }
}
