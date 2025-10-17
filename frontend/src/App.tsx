import React from 'react'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import { StylesProvider } from '@mui/styles'
import CssBaseline from '@mui/material/CssBaseline'
import ComponentStatusList from './components/ComponentStatusList'
import Header from './components/Header'

const theme = createTheme()

function App() {
  return (
    <StylesProvider injectFirst>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Header />
        <ComponentStatusList />
      </ThemeProvider>
    </StylesProvider>
  )
}

export default App
