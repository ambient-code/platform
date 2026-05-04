import * as coderabbitApi from '../../api/coderabbit-auth'
import type { CodeRabbitPort } from '../../ports/coderabbit'

type CodeRabbitApi = typeof coderabbitApi

export function createCodeRabbitAdapter(api: CodeRabbitApi): CodeRabbitPort {
  return {
    getCodeRabbitStatus: api.getCodeRabbitStatus,
    connectCodeRabbit: api.connectCodeRabbit,
    disconnectCodeRabbit: api.disconnectCodeRabbit,
  }
}

export const coderabbitAdapter = createCodeRabbitAdapter(coderabbitApi)
