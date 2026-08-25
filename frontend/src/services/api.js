import axios from 'axios';

// Create axios instance
const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080/api',
  timeout: 10000,
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors
api.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// Building API
export const buildingsAPI = {
  // Get all buildings
  getAll: () => api.get('/buildings'),
  
  // Get building by ID
  getById: (id) => api.get(`/buildings/${id}`),
  
  // Create building (admin only)
  create: (buildingData) => api.post('/buildings', buildingData),
};

// Lost & Found Areas API
export const areasAPI = {
  // Get all areas
  getAll: () => api.get('/areas'),
  
  // Get areas by building
  getByBuilding: (buildingId) => api.get(`/areas/building/${buildingId}`),
  
  // Create area (admin only)
  create: (areaData) => api.post('/areas', areaData),
};

// User API
export const usersAPI = {
  // Get user by ID
  getById: (id) => api.get(`/users/${id}`),
};

// Auth API
export const authAPI = {
  // Sign in with a Google ID token (from Google Identity Services)
  googleLogin: (credential) => api.post('/auth/google', { credential }),

  // Development-only login (backend mounts this only when ENVIRONMENT=development)
  devLogin: (email, name) => api.post('/auth/dev-login', { email, name }),
};

// Posts API
export const postsAPI = {
  // Search posts
  search: (params) => {
    const queryParams = new URLSearchParams();
    
    if (params.lat) queryParams.append('lat', params.lat);
    if (params.lng) queryParams.append('lng', params.lng);
    if (params.radius) queryParams.append('radius', params.radius);
    if (params.type) queryParams.append('type', params.type);
    if (params.category) queryParams.append('category', params.category);
    if (params.building_id) queryParams.append('building_id', params.building_id);
    if (params.lost_found_area_id) queryParams.append('lost_found_area_id', params.lost_found_area_id);
    if (params.is_lost_item !== undefined) queryParams.append('is_lost_item', params.is_lost_item);
    if (params.limit) queryParams.append('limit', params.limit);
    if (params.offset) queryParams.append('offset', params.offset);
    
    return api.get(`/posts?${queryParams.toString()}`);
  },
  
  // Get post by ID
  getById: (id) => api.get(`/posts/${id}`),
  
  // Create post
  create: (formData) => api.post('/posts', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  }),
  
  // Update post (requires the post's edit token)
  update: (id, postData, editToken) => api.put(`/posts/${id}`, postData, {
    headers: { 'X-Edit-Token': editToken || getEditToken(id) || '' },
  }),

  // Delete post (requires the post's edit token)
  delete: (id, editToken) => api.delete(`/posts/${id}`, {
    headers: { 'X-Edit-Token': editToken || getEditToken(id) || '' },
  }),
  
  // Claim post
  claim: (id) => api.post(`/posts/${id}/claim`),
};

// Interactions API (claims / help offers on a post)
export const interactionsAPI = {
  // Anyone can submit a claim/help interaction with their contact info
  create: (postId, data) => api.post(`/posts/${postId}/interactions`, data),

  // Only the poster (edit token holder) can list interactions on their post
  listForPost: (postId, editToken) => api.get(`/posts/${postId}/interactions`, {
    headers: { 'X-Edit-Token': editToken || getEditToken(postId) || '' },
  }),

  // Only the poster can accept/reject an interaction
  updateStatus: (interactionId, status, postId, editToken) => api.put(`/interactions/${interactionId}`, { status }, {
    headers: { 'X-Edit-Token': editToken || getEditToken(postId) || '' },
  }),
};

// Auth utilities
export const saveAuthToken = (token) => {
  localStorage.setItem('auth_token', token);
};

export const getAuthToken = () => {
  return localStorage.getItem('auth_token');
};

export const removeAuthToken = () => {
  localStorage.removeItem('auth_token');
};

// Edit tokens are stored per post so users can manage every post they created
const EDIT_TOKENS_KEY = 'edit_tokens';

const readEditTokens = () => {
  try {
    return JSON.parse(localStorage.getItem(EDIT_TOKENS_KEY)) || {};
  } catch {
    return {};
  }
};

export const saveEditToken = (postId, token) => {
  const tokens = readEditTokens();
  tokens[postId] = token;
  localStorage.setItem(EDIT_TOKENS_KEY, JSON.stringify(tokens));
};

export const getEditToken = (postId) => {
  return readEditTokens()[postId] || null;
};

export const removeEditToken = (postId) => {
  const tokens = readEditTokens();
  delete tokens[postId];
  localStorage.setItem(EDIT_TOKENS_KEY, JSON.stringify(tokens));
};

// Sign-in helpers: store the session token + user returned by the backend
const storeSession = (data) => {
  saveAuthToken(data.token);
  localStorage.setItem('user', JSON.stringify(data.user));
  return data.user;
};

export const loginWithGoogle = async (credential) => {
  const response = await authAPI.googleLogin(credential);
  if (response.success) {
    return storeSession(response.data);
  }
  throw new Error(response.error || 'Failed to sign in');
};

export const loginDev = async (email, name) => {
  const response = await authAPI.devLogin(email, name);
  if (response.success) {
    return storeSession(response.data);
  }
  throw new Error(response.error || 'Failed to sign in');
};

export const getCurrentUser = () => {
  const userStr = localStorage.getItem('user');
  if (userStr) {
    try {
      return JSON.parse(userStr);
    } catch (error) {
      console.error('Failed to parse user data:', error);
      return null;
    }
  }
  return null;
};

export const logout = () => {
  localStorage.removeItem('user');
  localStorage.removeItem('auth_token');
  // Edit tokens are intentionally kept: they prove ownership of posts
  // independently of login state.
};

export default api; 