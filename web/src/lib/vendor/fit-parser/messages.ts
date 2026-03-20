import type { MessageObject } from './fit.js'
import type { MesgNum } from './fit_types.js'
import { FIT } from './fit.js'

function getFieldObject(
  fieldNum: number,
  messageNum: number,
): MessageObject {
  const message = FIT.messages[messageNum]
  if (!message) {
    return {} as MessageObject
  }
  return message[fieldNum] || {}
}

function getMessageName(messageNum: number): MesgNum | '' {
  const message = FIT.messages[messageNum]
  return message ? message.name : ''
}

export function getFitMessage(messageNum: number): { name: MesgNum | '', getAttributes: (fieldNum: number) => MessageObject } {
  return {
    name: getMessageName(messageNum),
    getAttributes: (fieldNum: number) => getFieldObject(fieldNum, messageNum),
  }
}

// TODO
export function getFitMessageBaseType(inp: any): any {
  return inp
}