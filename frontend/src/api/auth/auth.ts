import axios from 'axios'

import { Tokens } from '../../types'

const apiHostname = process.env.REACT_APP_API_HOSTNAME

const authProvider = {
  isAuthenticated: false,
  signin(
    username: string,
    password: string,
    callback: (error: any, tokens: Tokens | null) => void
  ) {
    axios.post(`${apiHostname}/auth/v1/login`, {
      username,
      password,
    })
      .then((response) => {
        authProvider.isAuthenticated = true
        callback(null, response.data.tokens)
      })
      .catch((error) => {
        if (error.response) {
          callback(error.response.data.error, null)
        } else {
          callback(error.message, null)
        }
      })
  },
  signout(accessToken: string, callback: (error: any) => void) {
    axios.post(`${apiHostname}/auth/v1/logout`, null, {
      headers: { Authorization: `Bearer ${accessToken}`}
    })
      .then(() => {
        authProvider.isAuthenticated = false
        callback(null)
      })
      .catch((error) => {
        if (error.response) {
          callback(error.response.data.error)
        } else {
          callback(error.message)
        }
      })
  },
}

export { authProvider }
