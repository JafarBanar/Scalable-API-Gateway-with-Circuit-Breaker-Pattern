use axum::{
    routing::{get, post},
    Router,
    Json,
    http::StatusCode,
    extract::Path,
};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
use redis::{Client, AsyncCommands, RedisError};
use std::sync::Arc;
use axum::serve;

#[derive(Debug, Serialize, Deserialize)]
struct CacheEntry {
    key: String,
    value: String,
    ttl: Option<u64>,
}

#[derive(Clone)]
struct AppState {
    redis_client: Arc<Client>,
}

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    // Initialize Redis client
    let redis_url = std::env::var("REDIS_URL").unwrap_or_else(|_| "redis://redis:6379".into());
    let redis_client = Client::open(redis_url).expect("Failed to create Redis client");
    let redis_client = Arc::new(redis_client);
    
    let state = AppState { redis_client };

    // Build our application with a route
    let app = Router::new()
        .route("/health", get(health_check))
        .route("/cache", post(set_cache))
        .route("/cache/:key", get(get_cache))
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    // Run it
    let addr = SocketAddr::from(([0, 0, 0, 0], 8081));
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    tracing::info!("listening on {}", addr);
    axum::serve(listener, app).await.unwrap();
}

async fn health_check() -> StatusCode {
    StatusCode::OK
}

async fn set_cache(
    axum::extract::State(state): axum::extract::State<AppState>,
    Json(payload): Json<CacheEntry>,
) -> StatusCode {
    let mut conn = match state.redis_client.get_async_connection().await {
        Ok(conn) => conn,
        Err(e) => {
            tracing::error!("Failed to connect to Redis: {}", e);
            return StatusCode::INTERNAL_SERVER_ERROR;
        }
    };

    let result = match payload.ttl {
        Some(ttl) => conn.set_ex::<_, _, ()>(&payload.key, &payload.value, ttl).await,
        None => conn.set::<_, _, ()>(&payload.key, &payload.value).await,
    };

    match result {
        Ok(_) => {
            tracing::info!("Successfully cached value for key: {}", payload.key);
            StatusCode::OK
        }
        Err(e) => {
            tracing::error!("Failed to set cache: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR
        }
    }
}

async fn get_cache(
    axum::extract::State(state): axum::extract::State<AppState>,
    Path(key): Path<String>,
) -> Result<Json<String>, StatusCode> {
    let mut conn = match state.redis_client.get_async_connection().await {
        Ok(conn) => conn,
        Err(e) => {
            tracing::error!("Failed to connect to Redis: {}", e);
            return Err(StatusCode::INTERNAL_SERVER_ERROR);
        }
    };

    match conn.get::<_, String>(&key).await {
        Ok(value) => {
            tracing::info!("Retrieved value for key: {}", key);
            Ok(Json(value))
        }
        Err(e) => {
            if e.to_string().contains("no such key") {
                tracing::info!("Key not found: {}", key);
                Err(StatusCode::NOT_FOUND)
            } else {
                tracing::error!("Failed to get cache: {}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }
} 