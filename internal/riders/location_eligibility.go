package riders

import "context"

func (s *Service) CanPublishLocation(ctx context.Context,userID string)(bool,error){
 profile,err:=s.Get(ctx,userID);if err!=nil{return false,err}
 return profile.VerificationStatus==VerificationApproved && (profile.Availability==AvailabilityAvailable||profile.Availability==AvailabilityBusy),nil
}
