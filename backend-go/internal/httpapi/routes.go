package httpapi

import (
	"net/http"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
)

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health/live", s.handleLive)
	s.mux.HandleFunc("GET /health/ready", s.handleReady)
	s.mux.HandleFunc("POST /v1/auth/dev/wechat-login", s.handleDevLogin)
	s.mux.Handle("GET /v1/auth/me", s.requireAuth(http.HandlerFunc(s.handleMe)))

	s.mux.Handle("POST /v1/demands", s.requireRoles(domain.RoleConsumer)(http.HandlerFunc(s.handleCreateDemand)))
	s.mux.Handle("GET /v1/demands", s.requireRoles(domain.RoleConsumer, domain.RoleMerchant, domain.RoleAdmin)(http.HandlerFunc(s.handleListDemands)))
	s.mux.Handle("GET /v1/demands/{id}", s.requireRoles(domain.RoleConsumer, domain.RoleMerchant, domain.RoleAdmin)(http.HandlerFunc(s.handleGetDemand)))
	s.mux.Handle("POST /v1/demands/{id}/join", s.requireRoles(domain.RoleConsumer)(http.HandlerFunc(s.handleJoinDemand)))
	s.mux.Handle("GET /v1/demands/{id}/members", s.requireRoles(domain.RoleMerchant, domain.RoleAdmin)(http.HandlerFunc(s.handleListDemandMembers)))

	s.mux.Handle("GET /v1/merchant/demands", s.requireRoles(domain.RoleMerchant)(http.HandlerFunc(s.handleMerchantDemands)))
	s.mux.Handle("POST /v1/merchant/offers", s.requireRoles(domain.RoleMerchant)(http.HandlerFunc(s.handleCreateOffer)))
	s.mux.Handle("GET /v1/merchant/offers", s.requireRoles(domain.RoleMerchant)(http.HandlerFunc(s.handleMerchantOffers)))
	s.mux.Handle("POST /v1/merchant/campaigns", s.requireRoles(domain.RoleMerchant)(http.HandlerFunc(s.handleCreateCampaign)))
	s.mux.Handle("GET /v1/merchant/campaigns", s.requireRoles(domain.RoleMerchant)(http.HandlerFunc(s.handleMerchantCampaigns)))

	s.mux.Handle("GET /v1/campaigns", s.requireRoles(domain.RoleConsumer, domain.RoleMerchant, domain.RoleAdmin)(http.HandlerFunc(s.handleListCampaigns)))
	s.mux.Handle("GET /v1/campaigns/{id}", s.requireRoles(domain.RoleConsumer, domain.RoleMerchant, domain.RoleAdmin)(http.HandlerFunc(s.handleGetCampaign)))
	s.mux.Handle("POST /v1/campaigns/{id}/orders", s.requireRoles(domain.RoleConsumer)(http.HandlerFunc(s.handleCreateOrder)))
	s.mux.Handle("GET /v1/orders", s.requireRoles(domain.RoleConsumer, domain.RoleAdmin)(http.HandlerFunc(s.handleListOrders)))
	s.mux.Handle("GET /v1/orders/{id}", s.requireRoles(domain.RoleConsumer, domain.RoleAdmin)(http.HandlerFunc(s.handleGetOrder)))

	s.mux.Handle("GET /v1/admin/demands", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleAdminDemands)))
	s.mux.Handle("PATCH /v1/admin/demands/{id}/review", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleReviewDemand)))
	s.mux.Handle("GET /v1/admin/campaigns", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleAdminCampaigns)))
	s.mux.Handle("PATCH /v1/admin/campaigns/{id}/review", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleReviewCampaign)))
	s.mux.Handle("GET /v1/admin/merchants", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleAdminMerchants)))
	s.mux.Handle("PATCH /v1/admin/merchants/{id}/review", s.requireRoles(domain.RoleAdmin)(http.HandlerFunc(s.handleReviewMerchant)))
}
