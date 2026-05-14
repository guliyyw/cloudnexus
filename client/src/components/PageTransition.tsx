import { useEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'

interface Props {
  children: React.ReactNode
}

export default function PageTransition({ children }: Props) {
  const location = useLocation()
  const [displayChildren, setDisplayChildren] = useState(children)
  const [transitionStage, setTransitionStage] = useState('fadeIn')
  const prevPath = useRef(location.pathname)

  useEffect(() => {
    if (location.pathname !== prevPath.current) {
      setTransitionStage('fadeOut')
      const timeout = setTimeout(() => {
        setDisplayChildren(children)
        setTransitionStage('fadeIn')
        prevPath.current = location.pathname
      }, 150)
      return () => clearTimeout(timeout)
    } else {
      setDisplayChildren(children)
    }
  }, [location.pathname, children])

  return (
    <div
      style={{
        opacity: transitionStage === 'fadeIn' ? 1 : 0,
        transform: transitionStage === 'fadeIn' ? 'translateY(0)' : 'translateY(8px)',
        transition: 'opacity 0.15s ease, transform 0.15s ease',
      }}
    >
      {displayChildren}
    </div>
  )
}
