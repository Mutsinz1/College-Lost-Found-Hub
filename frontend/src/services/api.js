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
  // Get or create user via SSO
  getOrCreate: (ssoUser) => api.post('/users/sso', ssoUser),
  
  // Get user by ID
  getById: (id) => api.get(`/users/${id}`),
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
  
  // Update post
  update: (id, postData) => api.put(`/posts/${id}`, postData),
  
  // Delete post
  delete: (id) => api.delete(`/posts/${id}`),
  
  // Claim post
  claim: (id) => api.post(`/posts/${id}/claim`),
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

export const saveEditToken = (token) => {
  localStorage.setItem('edit_token', token);
};

export const getEditToken = () => {
  return localStorage.getItem('edit_token');
};

export const removeEditToken = () => {
  localStorage.removeItem('edit_token');
};

// SSO utilities
export const handleSSOLogin = async (ssoUser) => {
  try {
    const response = await usersAPI.getOrCreate(ssoUser);
    if (response.success) {
      // Save user info to localStorage
      localStorage.setItem('user', JSON.stringify(response.data.user));
      return response.data.user;
    }
    throw new Error(response.error || 'Failed to authenticate');
  } catch (error) {
    console.error('SSO login error:', error);
    throw error;
  }
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
  localStorage.removeItem('edit_token');
};

// Mock SSO for development (replace with actual SSO integration)
export const mockSSOLogin = async () => {
  const mockUser = {
    sso_id: 'mock_sso_123',
    email: 'student@college.edu',
    name: 'John Student',
  };
  
  return await handleSSOLogin(mockUser);
};

export default api; 