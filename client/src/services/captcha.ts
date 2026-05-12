import api from './api'

export interface CaptchaData {
  captcha_id: string
  image_base64: string
}

export async function getCaptcha(): Promise<CaptchaData> {
  const res = await api.get('/user/captcha')
  return res.data.data
}

export async function verifyCaptcha(captchaId: string, code: string): Promise<boolean> {
  const res = await api.post('/user/captcha/verify', {
    captcha_id: captchaId,
    captcha_code: code,
  })
  return res.data.data.valid
}
