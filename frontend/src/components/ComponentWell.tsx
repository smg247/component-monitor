import React from 'react'
import { Box, Card, CardContent, Typography, styled } from '@mui/material'
import { Component, SubComponent } from '../types'
import SubComponentCard from './SubComponentCard'
import { StatusChip } from './StatusColors'
import { getStatusBackgroundColor } from '../helpers'

const ComponentWell = styled(Card)<{ status: string }>(({ theme, status }) => {
  const color = getStatusBackgroundColor(theme, status)

  return {
    backgroundColor: color,
    border: `2px solid ${color}`,
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

const HeaderBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  marginBottom: theme.spacing(2),
}))

const DescriptionTypography = styled(Typography)(({ theme }) => ({
  marginBottom: theme.spacing(2),
}))

interface ComponentWellProps {
  component: Component
}

const ComponentWellComponent: React.FC<ComponentWellProps> = ({ component }) => {
  return (
    <ComponentWell status={component.status || 'Unknown'}>
      <CardContent>
        <HeaderBox>
          <Typography variant="h5" component="h2">
            {component.name}
          </Typography>
          <StatusChip
            label={component.status || 'Unknown'}
            status={component.status || 'Unknown'}
            variant="filled"
          />
        </HeaderBox>

        <DescriptionTypography variant="body2" color="text.secondary">
          {component.description}
        </DescriptionTypography>

        <SubComponentsGrid>
          {component.sub_components.map((subComponent: SubComponent) => (
            <SubComponentCard
              key={subComponent.name}
              subComponent={subComponent}
              componentName={component.name}
            />
          ))}
        </SubComponentsGrid>
      </CardContent>
    </ComponentWell>
  )
}

export default ComponentWellComponent
