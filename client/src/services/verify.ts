import api from './api'

export async function sendEmailCode(email: string): Promise<void> {
  await api.post('/user/email/send-code', { email })
}

export async function verifyEmail(email: string, code: string): Promise<void> {
  await api.post('/user/email/verify', { email, code })
}

export async function sendPhoneCode(phone: string): Promise<void> {
  await api.post('/user/phone/send-code', { phone })
}

export async function verifyPhone(phone: string, code: string): Promise<void> {
  await api.post('/user/phone/verify', { phone, code })
}
