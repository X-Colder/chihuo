package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

func (s *Server) handleLive(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "NOT_READY", "store is not ready", nil)
		return
	}
	writeData(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !s.config.DevLoginEnabled {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "dev login is disabled", nil)
		return
	}
	s.handleLogin(w, r, DevWeChatLoginProvider{}, true)
}

func (s *Server) handleWeChatLogin(w http.ResponseWriter, r *http.Request) {
	if s.config.WeChatAppID == "" || s.config.WeChatAppSecret == "" {
		writeError(w, r, http.StatusNotFound, "WECHAT_LOGIN_NOT_CONFIGURED", "WeChat login is not configured", nil)
		return
	}
	s.handleLogin(w, r, s.provider, false)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request, provider WeChatLoginProvider, dev bool) {
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	var input devLoginRequest
	if err := decodeJSON(body, &input); err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if input.Role == "" {
		input.Role = domain.RoleConsumer
	}
	if !dev && input.Role == domain.RoleAdmin {
		s.writeOperationError(w, r, newRequestError(http.StatusForbidden, "INVALID_ROLE", "admin role cannot use WeChat client login", nil))
		return
	}
	if err := input.validate(); err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	identity, err := provider.Login(r.Context(), strings.TrimSpace(input.Code))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	user, err := s.store.CreateOrGetUser(r.Context(), store.CreateUserInput{
		Name:        strings.TrimSpace(input.Name),
		Role:        input.Role,
		ExternalKey: string(input.Role) + ":" + identity.Subject,
	})
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	session := domain.SessionUser{ID: user.ID, Name: user.Name, Role: user.Role}
	if user.Role == domain.RoleMerchant {
		merchantName := strings.TrimSpace(input.MerchantName)
		if merchantName == "" {
			merchantName = user.Name + "的店"
		}
		merchant, merchantErr := s.store.CreateOrGetMerchant(r.Context(), store.CreateMerchantInput{
			OwnerUserID: user.ID,
			Name:        merchantName,
			License:     map[string]any{"verification_status": "PENDING"},
		})
		if merchantErr != nil {
			s.writeOperationError(w, r, merchantErr)
			return
		}
		session.MerchantID = merchant.ID
	}
	token, err := s.signer.Sign(session, timeNow())
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  session,
		"mode":  map[bool]string{true: "dev", false: "wechat"}[dev],
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	current, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication is required", nil)
		return
	}
	writeData(w, http.StatusOK, domain.SessionUser{
		ID:         current.ID,
		Name:       current.Name,
		Role:       current.Role,
		MerchantID: current.MerchantID,
	})
}

func (s *Server) handleCreateDemand(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input createDemandRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if input.MinimumMembers == 0 {
			input.MinimumMembers = 10
		}
		if input.MaximumMembers == 0 {
			input.MaximumMembers = 100
		}
		spec, err := input.toSpec()
		if err != nil {
			return 0, nil, err
		}
		matching, matchErr := s.store.FindMatchingDemand(r.Context(), spec)
		if matchErr == nil {
			member, demand, addErr := s.store.AddDemandMember(r.Context(), store.CreateMemberInput{
				DemandID:    matching.ID,
				UserID:      current.ID,
				Quantity:    input.Quantity,
				WeightGrams: exactWeight(input.WeightMinGrams, input.WeightMaxGrams),
				Preferences: spec.Preferences,
				Notes:       spec.Notes,
			})
			if addErr != nil {
				return 0, nil, addErr
			}
			return http.StatusOK, map[string]any{"demand": demand, "member": member, "matched": true}, nil
		}
		if !errors.Is(matchErr, store.ErrNotFound) {
			return 0, nil, matchErr
		}
		demand, createErr := s.store.CreateDemand(r.Context(), store.CreateDemandInput{
			CreatedBy:      current.ID,
			MinimumMembers: input.MinimumMembers,
			MaximumMembers: input.MaximumMembers,
			Spec:           spec,
		})
		if createErr != nil {
			return 0, nil, createErr
		}
		member, demand, addErr := s.store.AddDemandMember(r.Context(), store.CreateMemberInput{
			DemandID:    demand.ID,
			UserID:      current.ID,
			Quantity:    input.Quantity,
			WeightGrams: exactWeight(input.WeightMinGrams, input.WeightMaxGrams),
			Preferences: spec.Preferences,
			Notes:       spec.Notes,
		})
		if addErr != nil {
			return 0, nil, addErr
		}
		return http.StatusCreated, map[string]any{"demand": demand, "member": member, "matched": false}, nil
	})
}

func (s *Server) handleListDemands(w http.ResponseWriter, r *http.Request) {
	options := listOptionsFromRequest(r)
	demands, err := s.store.ListDemands(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, demands)
}

func (s *Server) handleAdminDemands(w http.ResponseWriter, r *http.Request) {
	options := listOptionsFromRequest(r)
	demands, err := s.store.ListDemands(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, demands)
}

func (s *Server) handleGetDemand(w http.ResponseWriter, r *http.Request) {
	demand, err := s.store.GetDemand(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, demand)
}

func (s *Server) handleJoinDemand(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input joinDemandRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if input.Quantity == 0 {
			input.Quantity = 1
		}
		if err := input.validate(); err != nil {
			return 0, nil, err
		}
		member, demand, err := s.store.AddDemandMember(r.Context(), store.CreateMemberInput{
			DemandID:    r.PathValue("id"),
			UserID:      current.ID,
			Quantity:    input.Quantity,
			WeightGrams: input.WeightGrams,
			Preferences: normalizeStrings(input.Preferences, 20, 80),
			Notes:       optionalText(input.Notes, 1000),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, map[string]any{"demand": demand, "member": member}, nil
	})
}

func (s *Server) handleListDemandMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.store.ListDemandMembers(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	for index := range members {
		members[index].UserID = ""
	}
	writeData(w, http.StatusOK, members)
}

func (s *Server) handleMerchantDemands(w http.ResponseWriter, r *http.Request) {
	options := listOptionsFromRequest(r)
	demands, err := s.store.ListDemands(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, demands)
}

func (s *Server) handleCreateOffer(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input createOfferRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(); err != nil {
			return 0, nil, err
		}
		if input.OilLevel == "" {
			input.OilLevel = "UNKNOWN"
		}
		if input.SaltLevel == "" {
			input.SaltLevel = "UNKNOWN"
		}
		offer, err := s.store.CreateOffer(r.Context(), store.CreateOfferInput{
			DemandID:            input.DemandID,
			MerchantID:          current.MerchantID,
			UnitPriceCents:      input.UnitPriceCents,
			ProductionCapacity:  input.ProductionCapacity,
			WeightGrams:         input.WeightGrams,
			Ingredients:         normalizeStrings(input.Ingredients, 100, 100),
			Allergens:           normalizeStrings(input.Allergens, 50, 100),
			OilLevel:            input.OilLevel,
			SaltLevel:           input.SaltLevel,
			ProductionTime:      input.ProductionTime,
			ShelfLifeMinutes:    input.ShelfLifeMinutes,
			StorageInstructions: strings.TrimSpace(input.StorageInstructions),
			Notes:               optionalText(input.Notes, 1000),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, offer, nil
	})
}

func (s *Server) handleMerchantOffers(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	options := listOptionsFromRequest(r)
	options.MerchantID = current.MerchantID
	offers, err := s.store.ListOffers(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, offers)
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input createCampaignRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(); err != nil {
			return 0, nil, err
		}
		if input.FoodSpec.OilLevel == "" {
			input.FoodSpec.OilLevel = "UNKNOWN"
		}
		if input.FoodSpec.SaltLevel == "" {
			input.FoodSpec.SaltLevel = "UNKNOWN"
		}
		campaign, err := s.store.CreateCampaign(r.Context(), store.CreateCampaignInput{
			DemandID:         input.DemandID,
			OfferID:          input.OfferID,
			MerchantID:       current.MerchantID,
			Title:            strings.TrimSpace(input.Title),
			Description:      optionalText(input.Description, 1000),
			UnitPriceCents:   input.UnitPriceCents,
			DeliveryFeeCents: input.DeliveryFeeCents,
			PlatformFeeBPS:   input.PlatformFeeBPS,
			MinimumOrders:    input.MinimumOrders,
			MaximumOrders:    input.MaximumOrders,
			StartsAt:         input.StartsAt,
			EndsAt:           input.EndsAt,
			PickupPoint:      strings.TrimSpace(input.PickupPoint),
			FoodSpec:         input.FoodSpec.toDomain(),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, campaign, nil
	})
}

func (s *Server) handleMerchantCampaigns(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	options := listOptionsFromRequest(r)
	options.MerchantID = current.MerchantID
	campaigns, err := s.store.ListCampaigns(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, campaigns)
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	options := listOptionsFromRequest(r)
	if current.Role == domain.RoleConsumer && options.Status == "" {
		options.Status = string(domain.CampaignOpen)
	}
	campaigns, err := s.store.ListCampaigns(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, campaigns)
}

func (s *Server) handleAdminCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := s.store.ListCampaigns(r.Context(), listOptionsFromRequest(r))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, campaigns)
}

func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	campaign, err := s.store.GetCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, campaign)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input createOrderRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(); err != nil {
			return 0, nil, err
		}
		order, err := s.store.CreateOrder(r.Context(), store.CreateOrderInput{
			CampaignID:      r.PathValue("id"),
			ConsumerID:      current.ID,
			Quantity:        input.Quantity,
			DeliveryAddress: strings.TrimSpace(input.DeliveryAddress),
			ContactName:     strings.TrimSpace(input.ContactName),
			ContactPhone:    strings.TrimSpace(input.ContactPhone),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, order, nil
	})
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	options := listOptionsFromRequest(r)
	if current.Role == domain.RoleConsumer {
		options.ConsumerID = current.ID
	}
	orders, err := s.store.ListOrders(r.Context(), options)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, orders)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	order, err := s.store.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if current.Role == domain.RoleConsumer && order.ConsumerID != current.ID {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "order does not belong to the current user", nil)
		return
	}
	writeData(w, http.StatusOK, order)
}

func (s *Server) handleReviewDemand(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input reviewRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(string(domain.DemandOpen), string(domain.DemandRejected)); err != nil {
			return 0, nil, err
		}
		demand, err := s.store.ReviewDemand(r.Context(), r.PathValue("id"), store.ReviewInput{
			Status:     input.Status,
			ReviewerID: current.ID,
			Reason:     optionalText(input.Reason, 500),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, demand, nil
	})
}

func (s *Server) handleReviewCampaign(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input reviewRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(string(domain.CampaignOpen), string(domain.CampaignCancelled)); err != nil {
			return 0, nil, err
		}
		campaign, err := s.store.ReviewCampaign(r.Context(), r.PathValue("id"), store.ReviewInput{
			Status:     input.Status,
			ReviewerID: current.ID,
			Reason:     optionalText(input.Reason, 500),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, campaign, nil
	})
}

func (s *Server) handleAdminMerchants(w http.ResponseWriter, r *http.Request) {
	merchants, err := s.store.ListMerchants(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, merchants)
}

func (s *Server) handleReviewMerchant(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input reviewRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		if err := input.validate(string(domain.MerchantApproved), string(domain.MerchantRejected), string(domain.MerchantSuspended)); err != nil {
			return 0, nil, err
		}
		merchant, err := s.store.ReviewMerchant(r.Context(), r.PathValue("id"), store.ReviewInput{
			Status:     input.Status,
			ReviewerID: current.ID,
			Reason:     optionalText(input.Reason, 500),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, merchant, nil
	})
}

func mustPrincipal(r *http.Request) principal {
	current, _ := principalFromContext(r.Context())
	return current
}

func exactWeight(minimum, maximum int) int {
	if minimum == maximum {
		return minimum
	}
	return 0
}

func listOptionsFromRequest(r *http.Request) store.ListOptions {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return store.ListOptions{
		Status:     strings.TrimSpace(query.Get("status")),
		DemandID:   strings.TrimSpace(query.Get("demand_id")),
		ConsumerID: strings.TrimSpace(query.Get("consumer_id")),
		Limit:      limit,
		Offset:     offset,
	}
}

func (v foodSpecDTO) toDomain() domain.FoodSpec {
	return domain.FoodSpec{
		WeightGrams:         v.WeightGrams,
		Ingredients:         normalizeStrings(v.Ingredients, 100, 100),
		Allergens:           normalizeStrings(v.Allergens, 50, 100),
		OilLevel:            v.OilLevel,
		SaltLevel:           v.SaltLevel,
		ProductionTime:      v.ProductionTime,
		ShelfLifeMinutes:    v.ShelfLifeMinutes,
		StorageInstructions: strings.TrimSpace(v.StorageInstructions),
	}
}
