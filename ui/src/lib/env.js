// eslint-disable-next-line import/prefer-default-export
export const WS_URI =
  process.env.REACT_APP_BACKEND_URI ? process.env.REACT_APP_BACKEND_URI :
    `ws${window.location.protocol === 'https:' ? 's' : ''}://${window.location.hostname === 'localhost' ? 'localhost:3000' : window.location.host}/stream`;
