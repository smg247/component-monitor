import React, { useState, useEffect } from 'react'
import { Box, Card, CardContent, Typography, styled } from '@mui/material'
import { StatusChip } from './StatusColors'
import { getStatusChipColor } from '../helpers'
import { SubComponent } from '../types'
import { getSubComponentStatusEndpoint } from '../endpoints'
import OutageModal from './OutageModal'

const SubComponentCard = styled(Card)<{ status: string }>(({ theme, status }) => {
  const color = getStatusChipColor(theme, status)

  return {
    border: `1px solid ${color}`,
    cursor: 'pointer',
    '&:hover': {
      boxShadow: theme.shadows[2],
    },
  }
})

const StyledCardContent = styled(CardContent)(({ theme }) => ({
  padding: theme.spacing(2),
}))

const StatusChipBox = styled(Box)(({ theme }) => ({
  marginTop: theme.spacing(1),
}))

interface SubComponentCardProps {
  subComponent: SubComponent
  componentName: string
}

const SubComponentCardComponent: React.FC<SubComponentCardProps> = ({
  subComponent,
  componentName,
}) => {
  const [modalOpen, setModalOpen] = useState(false)
  const [subComponentWithStatus, setSubComponentWithStatus] = useState<SubComponent>(subComponent)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch(getSubComponentStatusEndpoint(componentName, subComponent.name))
      .then((res) => res.json())
      .catch(() => ({ status: 'Unknown', active_outages: [] }))
      .then((subStatus) => {
        setSubComponentWithStatus({
          ...subComponent,
          status: subStatus.status,
          active_outages: subStatus.active_outages,
        })
      })
      .finally(() => {
        setLoading(false)
      })
  }, [componentName, subComponent])

  const handleClick = () => {
    setModalOpen(true)
  }

  const handleCloseModal = () => {
    setModalOpen(false)
  }

  return (
    <>
      <SubComponentCard status={subComponentWithStatus.status || 'Unknown'} onClick={handleClick}>
        <StyledCardContent>
          <Typography variant="subtitle2" gutterBottom>
            {subComponent.name}
          </Typography>
          <Typography variant="caption" display="block" color="text.secondary">
            {subComponent.description}
          </Typography>
          <StatusChipBox>
            <StatusChip
              label={loading ? 'Loading...' : subComponentWithStatus.status || 'Unknown'}
              status={subComponentWithStatus.status || 'Unknown'}
              size="small"
              variant="outlined"
            />
          </StatusChipBox>
        </StyledCardContent>
      </SubComponentCard>

      <OutageModal
        open={modalOpen}
        onClose={handleCloseModal}
        selectedSubComponent={subComponentWithStatus}
        componentName={componentName}
      />
    </>
  )
}

export default SubComponentCardComponent
