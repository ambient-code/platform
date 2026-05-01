import type { CodeRabbitStatus, CodeRabbitConnectRequest } from './types'

export type CodeRabbitPort = {
  getCodeRabbitStatus: () => Promise<CodeRabbitStatus>
  connectCodeRabbit: (data: CodeRabbitConnectRequest) => Promise<void>
  disconnectCodeRabbit: () => Promise<void>
}
