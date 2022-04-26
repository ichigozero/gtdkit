import Paper from '@mui/material/Paper'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import axios from 'axios'
import * as React from 'react'

import useAuth from '../../hooks/useAuth'

const apiHostname = process.env.REACT_APP_API_HOSTNAME

interface Task {
  id: string
  title: string
  description: string
  done: string
  userId: number
}

const HomePage = () => {
  const { tokens } = useAuth()
  const [tasks, setTasks] = React.useState<Task[] | null>(null)

  React.useEffect(() => {
    if (!tokens) return

    axios.get(`${apiHostname}/task/v1/tasks`, {
      headers: { Authorization: `Bearer ${tokens.access}`}
    })
      .then((response) => {
        setTasks(response.data.tasks)
      })
  }, [tokens])

  if (!tasks) {
    return null
  }

  return (
    <TableContainer component={Paper}>
      <Table sx={{ minWidth: 650 }} aria-label="simple table">
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Title</TableCell>
            <TableCell>Description</TableCell>
            <TableCell>Done</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {tasks.map((row) => (
            <TableRow key={row.id}>
              <TableCell component="th" scope="row">
                {row.id}
              </TableCell>
              <TableCell>{row.title}</TableCell>
              <TableCell>{row.description}</TableCell>
              <TableCell>{row.done}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default HomePage
