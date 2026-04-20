import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

export interface ParsedEvent {
  title: string;
  description: string;
  event_date: string; // YYYY-MM-DD or empty
  note: string;       // original time reference
}

export interface ParsedTask {
  title: string;
  description: string;
  date: string; // YYYY-MM-DD; today when empty
}

export interface ParsedGoal {
  title: string;
  description: string;
  weight: number;
  tags: string[];
  timeline: string;
  extracted_events: ParsedEvent[];
  extracted_tasks: ParsedTask[];
}

// Goals
export const getGoals = () => api.get('/api/goals').then(r => r.data);
export const createGoal = (data: object) => api.post('/api/goals', data).then(r => r.data);
export const updateGoal = (id: number, data: object) => api.put(`/api/goals/${id}`, data).then(r => r.data);
export const deleteGoal = (id: number) => api.delete(`/api/goals/${id}`).then(r => r.data);
export const parseGoal = (rawInput: string) =>
  api.post('/api/goals/parse', { raw_input: rawInput }).then(r => r.data as ParsedGoal);


export const createEventQuick = (data: { title: string; description?: string; event_date: string }) =>
  api.post('/api/events/quick', data).then(r => r.data);

export const createTaskQuick = (data: { title: string; description?: string; date?: string; goal_id?: number }) =>
  api.post('/api/plan/tasks/quick', data).then(r => r.data);

// Tasks
export const getTodayPlan = () => api.get('/api/plan/today').then(r => r.data);
export const updateTask = (id: number, data: object) => api.put(`/api/tasks/${id}`, data).then(r => r.data);
export const deleteTask = (id: number) => api.delete(`/api/tasks/${id}`).then(r => r.data);
export const generatePlan = () => api.post('/api/plan/generate').then(r => r.data);

export interface AgendaTask {
  id: number;
  title: string;
  description?: string;
  suggested_start?: string;
  suggested_end?: string;
  status: string;
  sort_order: number;
}

export interface AgendaGoal {
  id: number;
  title: string;
  description?: string;
  weight: number;
  tags: string;
  status: string;
  timeline: string;
}

export interface AgendaEvent {
  id: number;
  title: string;
  description?: string;
  event_date: string;
  status: string;
}

export interface DayAgenda {
  date: string;
  goals: AgendaGoal[];
  events: AgendaEvent[];
  tasks: AgendaTask[];
}

export interface WeekAgenda {
  days: DayAgenda[];
  unscheduled: AgendaGoal[];
}

export const getWeekAgenda = (start?: string) =>
  api.get('/api/agenda/week', { params: start ? { start } : {} }).then(r => r.data as WeekAgenda);

// Check-in
export const getTodayCheckin = () => api.get('/api/checkin/today').then(r => r.data);
export const submitCheckin = (data: object) => api.post('/api/checkin', data).then(r => r.data);

// Events
export const getEvents = () => api.get('/api/events').then(r => r.data);
export const createEvent = (data: FormData | object) => {
  if (data instanceof FormData) {
    return api.post('/api/events', data, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data);
  }
  return api.post('/api/events', data).then(r => r.data);
};
export const deleteEvent = (id: number) => api.delete(`/api/events/${id}`).then(r => r.data);
export const submitEventRetro = (id: number, data: object) => api.post(`/api/events/${id}/retro`, data).then(r => r.data);

// Auth
export const authRegister = (data: { username: string; password: string; email: string }) =>
  api.post('/api/auth/register', data).then(r => r.data);
export const authLogin = (data: { username: string; password: string }) =>
  api.post('/api/auth/login', data).then(r => r.data);
export const authLogout = () => api.post('/api/auth/logout').then(r => r.data);
export const authGetMe = () =>
  api.get('/api/auth/me').then(r => r.data).catch((err) => {
    if (err?.response?.status === 401) return null;
    throw err;
  });
export const authVerifyEmail = (token: string) =>
  api.get(`/api/auth/verify-email?token=${encodeURIComponent(token)}`).then(r => r.data);
export const authResendVerification = (email: string) =>
  api.post('/api/auth/resend-verification', { email }).then(r => r.data);
export const authForgotPassword = (email: string) =>
  api.post('/api/auth/forgot-password', { email }).then(r => r.data);
export const authResetPassword = (token: string, password: string) =>
  api.post('/api/auth/reset-password', { token, password }).then(r => r.data);

// User profile
export const updateUserProfile = (data: FormData) =>
  api.post('/api/user/profile', data, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data);
