import React from 'react'
import {
  Box,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemText,
  Divider,
  styled,
} from '@mui/material'
import { StatusChip, SeverityChip } from './StatusColors'
import { SubComponent } from '../types'

const StyledDialog = styled(Dialog)(({ theme }) => ({
  '& .MuiDialog-paper': {
    borderRadius: theme.spacing(2),
  },
}))

const HeaderBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
}))

const DescriptionTypography = styled(Typography)(({ theme }) => ({
  marginBottom: theme.spacing(2),
}))

const StyledListItem = styled(ListItem)(({ theme }) => ({
  paddingLeft: 0,
  paddingRight: 0,
}))

const OutageHeaderBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  gap: theme.spacing(1),
  marginBottom: theme.spacing(1),
}))

const TriageNotesTypography = styled(Typography)(({ theme }) => ({
  marginTop: theme.spacing(1),
}))

const NoOutagesBox = styled(Box)(({ theme }) => ({
  textAlign: 'center',
  paddingTop: theme.spacing(4),
  paddingBottom: theme.spacing(4),
}))

interface OutageModalProps {
  open: boolean
  onClose: () => void
  selectedSubComponent: SubComponent | null
  componentName?: string
}

const OutageModal: React.FC<OutageModalProps> = ({
  open,
  onClose,
  selectedSubComponent,
  componentName,
}) => {
  return (
    <StyledDialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      {selectedSubComponent && (
        <>
          <DialogTitle>
            <HeaderBox>
              <Typography variant="h6">
                {componentName} / {selectedSubComponent.name}
              </Typography>
              <StatusChip
                label={selectedSubComponent.status || 'Unknown'}
                status={selectedSubComponent.status || 'Unknown'}
                variant="filled"
              />
            </HeaderBox>
          </DialogTitle>
          <DialogContent>
            <DescriptionTypography variant="body2" color="text.secondary">
              {selectedSubComponent.description}
            </DescriptionTypography>

            {selectedSubComponent.active_outages &&
            selectedSubComponent.active_outages.length > 0 ? (
              <Box>
                <Typography variant="h6" gutterBottom>
                  Active Outages ({selectedSubComponent.active_outages.length})
                </Typography>
                <List>
                  {selectedSubComponent.active_outages.map((outage: any, index: number) => (
                    <React.Fragment key={outage.id}>
                      <StyledListItem alignItems="flex-start">
                        <ListItemText
                          primary={
                            <OutageHeaderBox>
                              <SeverityChip
                                label={outage.severity}
                                severity={outage.severity}
                                size="small"
                                variant="outlined"
                              />
                              <Typography variant="subtitle2">
                                {outage.description || 'No description'}
                              </Typography>
                            </OutageHeaderBox>
                          }
                          secondary={
                            <Box>
                              <Typography variant="caption" display="block">
                                Started: {new Date(outage.start_time).toLocaleString()}
                              </Typography>
                              {outage.discovered_by && (
                                <Typography variant="caption" display="block">
                                  Discovered by: {outage.discovered_by}
                                </Typography>
                              )}
                              {outage.triage_notes && (
                                <TriageNotesTypography variant="caption" display="block">
                                  Triage Notes: {outage.triage_notes}
                                </TriageNotesTypography>
                              )}
                            </Box>
                          }
                        />
                      </StyledListItem>
                      {index < (selectedSubComponent.active_outages?.length || 0) - 1 && (
                        <Divider />
                      )}
                    </React.Fragment>
                  ))}
                </List>
              </Box>
            ) : (
              <NoOutagesBox>
                <Typography variant="h6" color="text.secondary">
                  No Active Outages
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  This sub-component is currently healthy
                </Typography>
              </NoOutagesBox>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={onClose}>Close</Button>
          </DialogActions>
        </>
      )}
    </StyledDialog>
  )
}

export default OutageModal
