import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

// Goals
export const getGoals = () => api.get('/api/goals').then(r => r.data);
export const createGoal = (data: FormData | object) => {
  if (data instanceof FormData) {
    return api.post('/api/goals', data, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data);
  }
  return api.post('/api/goals', data).then(r => r.data);
};
export const updateGoal = (id: number, data: object) => api.put(`/api/goals/${id}`, data).then(r => r.data);
export const deleteGoal = (id: number) => api.delete(`/api/goals/${id}`).then(r => r.data);
export const setPrimaryGoal = (id: number) => api.put(`/api/goals/${id}/primary`).then(r => r.data);

// Tasks
export const getTodayPlan = () => api.get('/api/plan/today').then(r => r.data);
export const updateTask = (id: number, data: object) => api.put(`/api/tasks/${id}`, data).then(r => r.data);
export const generatePlan = () => api.post('/api/plan/generate').then(r => r.data);

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
