import * as React from 'react'

import { Tokens } from '../types'

interface AuthContextType {
  tokens: Tokens | null
  signin: (username: string, password: string, callback: (error: any) => void) => void
  signout: (callback: (error: any) => void) => void
}

const AuthContext = React.createContext<AuthContextType>(null!)

export default AuthContext
