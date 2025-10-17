import React, { useState, useEffect } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Chip,
  CircularProgress,
  Alert,
  Container,
  styled,
} from '@mui/material'
import { Component, ComponentStatus, SubComponentStatus } from './types'

const StyledContainer = styled(Container)(({ theme }) => ({
  marginTop: theme.spacing(4),
}))

const LoadingBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  justifyContent: 'center',
  alignItems: 'center',
  minHeight: '200px',
}))

const ComponentsGrid = styled(Box)(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))',
  gap: theme.spacing(3),
}))

const ComponentWell = styled(Card)<{ status: string }>(({ theme, status }) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Healthy':
        return theme.palette.success.light
      case 'Degraded':
        return theme.palette.warning.light
      case 'Down':
        return theme.palette.error.light
      case 'Suspected':
        return theme.palette.info.light
      case 'Partial':
        return theme.palette.secondary.light
      default:
        return theme.palette.grey[100]
    }
  }

  return {
    backgroundColor: getStatusColor(status),
    border: `2px solid ${getStatusColor(status)}`,
    '&:hover': {
      boxShadow: theme.shadows[4],
    },
  }
})

const SubComponentsGrid = styled(Box)(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
  gap: theme.spacing(2),
  marginTop: theme.spacing(2),
}))

const SubComponentCard = styled(Card)<{ status: string }>(({ theme, status }) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Healthy':
        return theme.palette.success.main
      case 'Degraded':
        return theme.palette.warning.main
      case 'Down':
        return theme.palette.error.main
      case 'Suspected':
        return theme.palette.info.main
      case 'Partial':
        return theme.palette.secondary.main
      default:
        return theme.palette.grey[500]
    }
  }

  return {
    border: `1px solid ${getStatusColor(status)}`,
    '&:hover': {
      boxShadow: theme.shadows[2],
    },
  }
})

interface ComponentWithSubStatuses extends Component {
  status: string
  subComponentStatuses: SubComponentStatus[]
}

const ComponentStatusList: React.FC = () => {
  const [components, setComponents] = useState<ComponentWithSubStatuses[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const apiBaseUrl = process.env.REACT_APP_API_BASE_URL

    // Fetch components configuration and their statuses
    Promise.all([
      fetch(`${apiBaseUrl}/api/components`).then((res) => res.json()),
      fetch(`${apiBaseUrl}/api/status`).then((res) => res.json()),
    ])
      .then(([componentsData, statusesData]) => {
        // Create a map of component statuses for quick lookup
        const statusMap = new Map<string, string>()
        statusesData.forEach((status: ComponentStatus) => {
          statusMap.set(status.component_name, status.status)
        })

        // Combine components with their statuses and fetch sub-component statuses
        const componentsWithStatuses = componentsData.map((component: Component) => ({
          ...component,
          status: statusMap.get(component.name) || 'Healthy',
        }))

        // Fetch sub-component statuses for each component
        const subComponentPromises = componentsWithStatuses.map((component: Component & { status: string }) =>
          Promise.all(
            component.sub_components.map((subComponent: any) =>
              fetch(`${apiBaseUrl}/api/status/${component.name}/${subComponent.name}`)
                .then((res) => res.json())
                .catch(() => ({ status: 'Healthy', active_outages: [] })),
            ),
          ).then((subStatuses) => ({
            ...component,
            subComponentStatuses: subStatuses,
          })),
        )

        return Promise.all(subComponentPromises)
      })
      .then((data) => {
        setComponents(data)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to fetch components')
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  if (loading) {
    return (
      <StyledContainer maxWidth="lg">
        <LoadingBox>
          <CircularProgress />
        </LoadingBox>
      </StyledContainer>
    )
  }

  if (error) {
    return (
      <StyledContainer maxWidth="lg">
        <Alert severity="error">{error}</Alert>
      </StyledContainer>
    )
  }

  return (
    <StyledContainer maxWidth="lg">
      <Typography variant="h4" component="h1" gutterBottom>
        Component Status Dashboard
      </Typography>

      <ComponentsGrid>
        {components.map((component) => (
          <ComponentWell key={component.name} status={component.status}>
            <CardContent>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  mb: 2,
                }}
              >
                <Typography variant="h5" component="h2">
                  {component.name}
                </Typography>
                <Chip
                  label={component.status}
                  color={
                    component.status === 'Healthy'
                      ? 'success'
                      : component.status === 'Degraded'
                        ? 'warning'
                        : component.status === 'Down'
                          ? 'error'
                          : component.status === 'Suspected'
                            ? 'info'
                            : 'secondary'
                  }
                  variant="filled"
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                {component.description}
              </Typography>

              <SubComponentsGrid>
                {component.sub_components.map((subComponent, index) => {
                  const subStatus = component.subComponentStatuses[index] || { status: 'Healthy' }
                  return (
                    <SubComponentCard key={subComponent.name} status={subStatus.status}>
                      <CardContent sx={{ p: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          {subComponent.name}
                        </Typography>
                        <Typography variant="caption" display="block" color="text.secondary">
                          {subComponent.description}
                        </Typography>
                        <Box sx={{ mt: 1 }}>
                          <Chip
                            label={subStatus.status}
                            size="small"
                            color={
                              subStatus.status === 'Healthy'
                                ? 'success'
                                : subStatus.status === 'Degraded'
                                  ? 'warning'
                                  : subStatus.status === 'Down'
                                    ? 'error'
                                    : subStatus.status === 'Suspected'
                                      ? 'info'
                                      : 'secondary'
                            }
                            variant="outlined"
                          />
                        </Box>
                      </CardContent>
                    </SubComponentCard>
                  )
                })}
              </SubComponentsGrid>
            </CardContent>
          </ComponentWell>
        ))}
      </ComponentsGrid>
    </StyledContainer>
  )
}

export default ComponentStatusList
