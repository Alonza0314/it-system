import ErrorBox from '../errorBox/errorBox'
import SuccessBox from '../successBox/successBox'

interface NotificationMessage {
  id: string
  message: string
  timestamp: number
}

interface NotificationContainerProps {
  errors: NotificationMessage[]
  successes: NotificationMessage[]
  onClose: (id: string) => void
}

export default function NotificationContainer({
  errors,
  successes,
  onClose,
}: NotificationContainerProps) {
  return (
    <>
      <ErrorBox
        errors={errors}
        onClose={onClose}
        duration={5000}
      />
      <SuccessBox
        successes={successes}
        onClose={onClose}
        duration={3000}
      />
    </>
  )
}
