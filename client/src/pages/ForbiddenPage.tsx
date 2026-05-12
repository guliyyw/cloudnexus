import { useNavigate } from 'react-router-dom'
import { Result, Button } from 'antd'

export default function ForbiddenPage() {
  const navigate = useNavigate()

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
      <Result
        status="403"
        title="403"
        subTitle="抱歉，你没有权限访问此页面"
        extra={
          <Button type="primary" onClick={() => navigate('/files')}>
            返回首页
          </Button>
        }
      />
    </div>
  )
}
