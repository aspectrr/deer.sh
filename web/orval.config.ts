import { defineConfig } from 'orval'

export default defineConfig({
  'fluid-api': {
    output: {
      client: 'react-query',
      mode: 'tags-split',
      clean: true,
      prettier: true,
      target: 'src/api',
      schemas: 'src/api/model',
      override: {
        operationName: (operation) => {
          return operation.operationId || ''
        },
        mutator: {
          path: './src/lib/axios.ts',
          name: 'customInstance',
        },
      },
    },
    input: {
      target: '../api/docs/openapi.yaml',
    },
  },
})
