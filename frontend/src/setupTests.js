import '@testing-library/jest-dom';

// jsdom has no geolocation; Home asks for the user's position on mount.
if (!global.navigator.geolocation) {
  global.navigator.geolocation = {
    getCurrentPosition: (success) =>
      success({ coords: { latitude: 40.7128, longitude: -74.006 } }),
    watchPosition: () => {},
    clearWatch: () => {},
  };
}

// Leaflet measures real layout, which jsdom does not do. These tests are about
// our components rendering and wiring, not about Leaflet's internals.
jest.mock('react-leaflet', () => ({
  MapContainer: ({ children }) => <div data-testid="map">{children}</div>,
  TileLayer: () => <div data-testid="tile-layer" />,
  Marker: ({ children }) => <div data-testid="marker">{children}</div>,
  Popup: ({ children }) => <div data-testid="popup">{children}</div>,
  useMapEvents: () => null,
  useMap: () => ({ setView: () => {} }),
}));
