// CRA's Jest config sets resetMocks: true, so mock implementations are wired
// up per test rather than once in beforeEach.
export const samplePost = (overrides = {}) => ({
  id: '11111111-1111-1111-1111-111111111111',
  type: 'found',
  category: 'item',
  title: 'Blue Jansport backpack',
  description: 'Found on a library table.',
  location: { latitude: 40.7128, longitude: -74.006 },
  status: 'active',
  is_lost_item: false,
  image_urls: [],
  created_at: '2026-08-01T12:00:00Z',
  expires_at: '2026-12-01T12:00:00Z',
  ...overrides,
});

export const apiMockFactory = () => ({
  postsAPI: {
    search: jest.fn(), getById: jest.fn(), create: jest.fn(),
    update: jest.fn(), delete: jest.fn(),
  },
  buildingsAPI: { getAll: jest.fn(), getById: jest.fn() },
  areasAPI: { getAll: jest.fn(), getByBuilding: jest.fn() },
  interactionsAPI: { create: jest.fn(), listForPost: jest.fn(), update: jest.fn() },
  saveEditToken: jest.fn(),
  getEditToken: jest.fn(),
  getCurrentUser: jest.fn(() => null),
  loginWithGoogle: jest.fn(),
  loginDev: jest.fn(),
  logout: jest.fn(),
});
