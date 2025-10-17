const getApiBaseUrl = () => process.env.REACT_APP_API_BASE_URL

export const getComponentsEndpoint = () => `${getApiBaseUrl()}/api/components`

export const getOverallStatusEndpoint = () => `${getApiBaseUrl()}/api/status`

export const getSubComponentStatusEndpoint = (componentName: string, subComponentName: string) =>
  `${getApiBaseUrl()}/api/status/${componentName}/${subComponentName}`
