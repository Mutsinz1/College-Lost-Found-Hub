import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import { MapPin, Search, Filter, Building, Map } from 'lucide-react';
import { postsAPI, buildingsAPI, areasAPI } from '../services/api';

const Home = () => {
  const [posts, setPosts] = useState([]);
  const [buildings, setBuildings] = useState([]);
  const [areas, setAreas] = useState([]);
  const [loading, setLoading] = useState(false);
  const [userLocation, setUserLocation] = useState(null);
  const [filters, setFilters] = useState({
    type: '',
    category: '',
    building_id: '',
    lost_found_area_id: '',
    is_lost_item: null,
    radius: 5000,
  });

  // Get user location on component mount
  useEffect(() => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          setUserLocation({
            lat: position.coords.latitude,
            lng: position.coords.longitude,
          });
        },
        (error) => {
          console.error('Error getting location:', error);
          // Default to a central location
          setUserLocation({ lat: 40.730610, lng: -73.935242 });
        }
      );
    } else {
      // Default to a central location
      setUserLocation({ lat: 40.730610, lng: -73.935242 });
    }
  }, []);

  // Load buildings and areas
  useEffect(() => {
    const loadBuildingsAndAreas = async () => {
      try {
        const [buildingsResponse, areasResponse] = await Promise.all([
          buildingsAPI.getAll(),
          areasAPI.getAll(),
        ]);
        
        if (buildingsResponse.success) {
          setBuildings(buildingsResponse.data.buildings);
        }
        
        if (areasResponse.success) {
          setAreas(areasResponse.data.areas);
        }
      } catch (error) {
        console.error('Failed to load buildings and areas:', error);
      }
    };

    loadBuildingsAndAreas();
  }, []);

  // Memoize searchPosts to avoid infinite re-renders
  const searchPosts = useCallback(async () => {
    if (!userLocation) return;

    setLoading(true);
    
    try {
      const params = {
        lat: userLocation.lat,
        lng: userLocation.lng,
        radius: filters.radius,
        ...(filters.type && { type: filters.type }),
        ...(filters.category && { category: filters.category }),
        ...(filters.building_id && { building_id: filters.building_id }),
        ...(filters.lost_found_area_id && { lost_found_area_id: filters.lost_found_area_id }),
        ...(filters.is_lost_item !== null && { is_lost_item: filters.is_lost_item }),
      };

      const response = await postsAPI.search(params);
      
      // Add null checks and default to empty array
      if (response && response.success && response.data) {
        setPosts(response.data.posts || []);
      } else {
        setPosts([]);
      }
    } catch (error) {
      console.error('Error searching posts:', error);
      // Don't show error toast for network issues, just set empty posts
      setPosts([]);
    } finally {
      setLoading(false);
    }
  }, [userLocation, filters]);

  // Search posts when location or filters change
  useEffect(() => {
    if (userLocation) {
      searchPosts();
    }
  }, [userLocation, filters, searchPosts]);

  const getMarkerIcon = (category) => {
    const icons = {
      pet: '🐕',
      document: '📄',
      item: '📱',
      other: '❓',
    };
    return icons[category] || '❓';
  };

  const getMarkerColor = (type, isLostItem) => {
    if (isLostItem) {
      return type === 'lost' ? '#ef4444' : '#10b981'; // Red for lost, green for found
    }
    return type === 'lost' ? '#f97316' : '#3b82f6'; // Orange for lost, blue for found
  };

  if (!userLocation) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="spinner"></div>
        <span className="ml-3 text-gray-600">Getting your location...</span>
      </div>
    );
  }

  // Ensure posts is always an array
  const safePosts = Array.isArray(posts) ? posts : [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          College Lost & Found
        </h1>
        <p className="text-gray-600">
          Find lost items or help others find what they've lost
        </p>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm border p-4">
        <div className="flex items-center space-x-2 mb-4">
          <Filter className="w-5 h-5 text-gray-500" />
          <h3 className="text-lg font-semibold text-gray-900">Filters</h3>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {/* Search Radius */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Search Radius
            </label>
            <select
              value={filters.radius}
              onChange={(e) => setFilters({ ...filters, radius: parseInt(e.target.value) })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value={1000}>1 km</option>
              <option value={5000}>5 km</option>
              <option value={10000}>10 km</option>
              <option value={25000}>25 km</option>
            </select>
          </div>

          {/* Type */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Type
            </label>
            <select
              value={filters.type}
              onChange={(e) => setFilters({ ...filters, type: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Types</option>
              <option value="lost">Lost</option>
              <option value="found">Found</option>
            </select>
          </div>

          {/* Category */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Category
            </label>
            <select
              value={filters.category}
              onChange={(e) => setFilters({ ...filters, category: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Categories</option>
              <option value="pet">Pet</option>
              <option value="document">Document</option>
              <option value="item">Item</option>
              <option value="other">Other</option>
            </select>
          </div>

          {/* Building */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Building
            </label>
            <select
              value={filters.building_id}
              onChange={(e) => setFilters({ ...filters, building_id: e.target.value, lost_found_area_id: '' })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Buildings</option>
              {buildings.map((building) => (
                <option key={building.id} value={building.id}>
                  {building.name}
                </option>
              ))}
            </select>
          </div>

          {/* Lost & Found Area */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Lost & Found Area
            </label>
            <select
              value={filters.lost_found_area_id}
              onChange={(e) => setFilters({ ...filters, lost_found_area_id: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
              disabled={!filters.building_id}
            >
              <option value="">All Areas</option>
              {areas
                .filter((area) => !filters.building_id || area.building_id === filters.building_id)
                .map((area) => (
                  <option key={area.id} value={area.id}>
                    {area.name} ({area.building?.name})
                  </option>
                ))}
            </select>
          </div>

          {/* Lost/Found Item Filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Item Type
            </label>
            <select
              value={filters.is_lost_item === null ? '' : filters.is_lost_item.toString()}
              onChange={(e) => {
                const value = e.target.value;
                setFilters({ 
                  ...filters, 
                  is_lost_item: value === '' ? null : value === 'true' 
                });
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Items</option>
              <option value="false">Found Items</option>
              <option value="true">Lost Items</option>
            </select>
          </div>
        </div>
      </div>

      {/* Map */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div className="p-4 border-b">
          <h2 className="text-lg font-semibold text-gray-900">
            Campus Map ({safePosts.length} items)
          </h2>
        </div>
        
        <div className="relative">
          <MapContainer
            center={[userLocation.lat, userLocation.lng]}
            zoom={15}
            style={{ height: '500px', width: '100%' }}
          >
            <TileLayer
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            />
            
            {/* User location marker */}
            <Marker position={[userLocation.lat, userLocation.lng]}>
              <Popup>
                <div className="text-center">
                  <MapPin className="w-4 h-4 mx-auto mb-1 text-primary-600" />
                  <span className="text-sm font-medium">Your Location</span>
                </div>
              </Popup>
            </Marker>

            {/* Building markers */}
            {buildings.map((building) => (
              <Marker
                key={building.id}
                position={[building.location.latitude, building.location.longitude]}
              >
                <Popup>
                  <div className="text-center">
                    <Building className="w-4 h-4 mx-auto mb-1 text-gray-600" />
                    <span className="text-sm font-medium">{building.name}</span>
                    {building.description && (
                      <p className="text-xs text-gray-500 mt-1">{building.description}</p>
                    )}
                  </div>
                </Popup>
              </Marker>
            ))}

            {/* Post markers */}
            {safePosts.map((post) => (
              <Marker
                key={post.id}
                position={[post.location.latitude, post.location.longitude]}
              >
                <Popup>
                  <div className="max-w-xs">
                    <div className="flex items-center space-x-2 mb-2">
                      <span className="text-lg">{getMarkerIcon(post.category)}</span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                        post.type === 'lost' 
                          ? 'bg-danger-100 text-danger-800' 
                          : 'bg-success-100 text-success-800'
                      }`}>
                        {post.is_lost_item ? 'Lost Item' : 'Found Item'}
                      </span>
                    </div>
                    <h3 className="font-semibold text-gray-900 mb-1">{post.title}</h3>
                    {post.description && (
                      <p className="text-sm text-gray-600 mb-2">{post.description}</p>
                    )}
                    {post.lost_found_area && (
                      <p className="text-xs text-gray-500 mb-2">
                        📍 {post.lost_found_area.name} ({post.lost_found_area.building?.name})
                      </p>
                    )}
                    <p className="text-xs text-gray-500 mb-2">
                      {Math.round(post.distance)}m away • {new Date(post.created_at).toLocaleDateString()}
                    </p>
                    <Link
                      to={`/post/${post.id}`}
                      className="block w-full text-center px-3 py-1 bg-primary-600 text-white text-xs rounded hover:bg-primary-700 transition-colors"
                    >
                      View Details{post.status === 'active' ? ' & Claim' : ''}
                    </Link>
                  </div>
                </Popup>
              </Marker>
            ))}
          </MapContainer>

          {loading && (
            <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center">
              <div className="spinner"></div>
            </div>
          )}
        </div>
      </div>

      {/* Posts List */}
      <div className="bg-white rounded-lg shadow-sm border">
        <div className="p-4 border-b">
          <h2 className="text-lg font-semibold text-gray-900">Recent Items</h2>
        </div>
        
        <div className="divide-y divide-gray-200">
          {safePosts.slice(0, 10).map((post) => (
            <div key={post.id} className="p-4 hover:bg-gray-50 transition-colors">
              <div className="flex items-start space-x-3">
                <div className="flex-shrink-0">
                  <span className="text-2xl">{getMarkerIcon(post.category)}</span>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2 mb-1">
                    <h3 className="text-sm font-medium text-gray-900 truncate">
                      {post.title}
                    </h3>
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                      post.type === 'lost' 
                        ? 'bg-danger-100 text-danger-800' 
                        : 'bg-success-100 text-success-800'
                    }`}>
                      {post.is_lost_item ? 'Lost Item' : 'Found Item'}
                    </span>
                    {post.status === 'claimed' && (
                      <span className="px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                        Claimed
                      </span>
                    )}
                  </div>
                  {post.description && (
                    <p className="text-sm text-gray-600 mb-2 line-clamp-2">
                      {post.description}
                    </p>
                  )}
                  {post.lost_found_area && (
                    <p className="text-xs text-gray-500 mb-1">
                      📍 {post.lost_found_area.name} ({post.lost_found_area.building?.name})
                    </p>
                  )}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-4 text-xs text-gray-500">
                      <span>{Math.round(post.distance)}m away</span>
                      <span>{new Date(post.created_at).toLocaleDateString()}</span>
                    </div>
                    <Link
                      to={`/post/${post.id}`}
                      className="px-3 py-1 bg-primary-600 text-white text-xs rounded hover:bg-primary-700 transition-colors"
                    >
                      View{post.status === 'active' ? ' & Claim' : ''}
                    </Link>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default Home;