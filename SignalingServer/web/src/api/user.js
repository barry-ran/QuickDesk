const API_BASE = '/api/v1'

export async function registerUser(data) {
  const response = await fetch(`${API_BASE}/user/register`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || '注册失败')
  }

  return response.json()
}

export async function loginUser(data) {
  const response = await fetch(`${API_BASE}/user/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || '登录失败')
  }

  return response.json()
}

export async function getUserInfo(token) {
  const response = await fetch(`${API_BASE}/user-list`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  })

  if (!response.ok) {
    throw new Error('获取用户信息失败')
  }

  return response.json()
}
