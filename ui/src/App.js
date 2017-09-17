import React, { Component } from 'react';
import { BrowserRouter, Redirect, Route, Switch } from 'react-router-dom';
// import logo from './logo.svg';
// import './App.css';
import Chatroom from './chat/Room';

class App extends Component {
  render() {
    // TODO grab the default general channel if unset
    return (
      <BrowserRouter>
        <Switch>
          <Route path="/messages/:channelID" component={Chatroom} />
          <Redirect from="/" to={`/messages/${process.env.REACT_APP_SLACK_CHANNEL || 'api-testing'}`} />
        </Switch>
      </BrowserRouter>
    );
  }
}

export default App;
