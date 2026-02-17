import Axios, { type AxiosRequestConfig } from 'axios'

export const axios = Axios.create({
  baseURL: import.meta.env.VITE_API_URL || '',
  withCredentials: true,
})

export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const promise = axios(config).then(({ data }) => data)
  return promise
}
