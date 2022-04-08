import * as React from 'react'

import AuthContext from '../../context/AuthContext'

const useAuth = () => {
  return React.useContext(AuthContext)
}

export default useAuth
